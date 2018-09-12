package staging

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/vektorprogrammet/build-system/nginx"
)

type Server struct {
	Repo       string
	Branch     string
	RootFolder string
	Domain     string
	UpdateProgress func(message string, progress int)
}

func (s *Server) Deploy() error {
	s.UpdateProgress("Creating server folder", 0)
	if err := s.createServerFolder(); err != nil {
		return err
	}

	s.UpdateProgress("Cloning repository", 10)
	if err := s.clone(); err != nil {
		return err
	}

	if err := s.checkout(); err != nil {
		return err
	}

	if err := s.createParametersFile(); err != nil {
		return err
	}

	s.UpdateProgress("Installing composer and NPM dependencies", 30)
	if err := s.install(); err != nil {
		return err
	}

	s.UpdateProgress("Creating database", 70)
	if err := s.createDatabase(); err != nil {
		return err
	}

	if err := s.setFolderPermissions(); err != nil {
		return err
	}

	s.UpdateProgress("Creating nginx instance", 85)
	if err := s.createNginxConfig(); err != nil {
		return err
	}

	s.UpdateProgress("Creating HTTPS certificate", 90)
	if err := s.secureWithHttps(); err != nil {
		return err
	}

	return nil
}

func (s *Server) Update() error {
	if err := s.runCommands([]string{
		fmt.Sprintf("git pull origin %s", s.Branch),
	}); err != nil {
		return err
	}

	if err := s.install(); err != nil {
		return err
	}

	return s.updateDatabase()
}

func (s *Server) Exists() bool {
	_, err := os.Stat(s.folder())

	return !os.IsNotExist(err)
}

func (s *Server) createServerFolder() error {
	if _, err := os.Stat(s.folder()); os.IsNotExist(err) {
		os.Mkdir(s.folder(), 0755)
	}

	return nil
}

func (s *Server) clone() error {
	return s.runCommand(fmt.Sprintf("git clone %s .", s.Repo))
}

func (s *Server) checkout() error {
	return s.runCommand(fmt.Sprintf("git checkout %s", s.Branch))
}

func (s *Server) setFolderPermissions() error {
	cacheDir := s.folder() + "/app/cache"
	logsDir := s.folder() + "/app/logs"
	imageDir := s.folder() + "/www/images"
	signatureDir := s.folder() + "/signatures"
	mediaDir := s.folder() + "/www/media"

	return s.runCommands([]string{
		"setfacl -R -m u:vektorprogrammet:rwX .",
		"setfacl -dR -m u:vektorprogrammet:rwX .",
		fmt.Sprintf("setfacl -R -m u:www-data:rwX %s %s %s %s %s", cacheDir, logsDir, imageDir, signatureDir, mediaDir),
		fmt.Sprintf("setfacl -dR -m u:www-data:rwX %s %s %s %s %s", cacheDir, logsDir, imageDir, signatureDir, mediaDir),
	})
}

func (s *Server) createNginxConfig() error {
	nginxConfig := nginx.Config{
		Root:       s.folder() + "/www",
		ServerName: s.ServerName(),
	}

	return s.runCommand(fmt.Sprintf("echo '%s' > /srv/nginx/%s", nginxConfig.String(), nginxConfig.ServerName))
}

func (s *Server) restartNginx() error {
	return s.runCommand(fmt.Sprintf("sudo service nginx restart"))
}

func (s *Server) secureWithHttps() error {
	return s.runCommand(fmt.Sprintf("printf '2\n' | sudo certbot --nginx -d %s", s.ServerName()))
}

func (s *Server) folder() string {
	return s.RootFolder + "/" + s.Branch
}

func (s *Server) ServerName() string {
	return strings.Replace(s.Branch, "_", "-", -1) + "." + s.Domain
}

func (s *Server) install() error {
	var wg sync.WaitGroup
	wg.Add(3)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		s.runCommand("SYMFONY_ENV=prod php ./composer.phar install -n --no-dev --optimize-autoloader")
	}(&wg)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		s.runCommands([]string{
			"npm install",
			"npm run build:prod",
		})
	}(&wg)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		s.runCommands([]string{
			"npm run setup:scheduling",
			"npm run build:scheduling",
		})
	}(&wg)

	wg.Wait()

	return s.runCommand("php app/console cache:clear --env=prod")
}

func (s *Server) createDatabase() error {
	commands := []string{
		"php app/console doctrine:database:create --env=prod",
		"php app/console doctrine:schema:create --env=prod",
		"php app/console doctrine:fixtures:load -n",
		"php app/console doctrine:migrations:version --add --all -n --env=prod",
	}

	return s.runCommands(commands)
}

func (s *Server) updateDatabase() error {

	return s.runCommand("php app/console doctrine:migrations:migrate -n --env=prod ")
}

func (s *Server) createParametersFile() error {
	if err := s.runCommand("cp ../parameters.yml app/config/parameters.yml"); err != nil {
		return err
	}

	return s.runCommand(fmt.Sprintf("sed -i 's/dbname/%s/g' app/config/parameters.yml", s.Branch))
}

func (s *Server) dropDatabase() error {
	return s.runCommand("php app/console doctrine:database:drop --force")
}

func (s *Server) remove() error {
	if len(s.folder()) > len(s.RootFolder) + 1 {
		defer s.runCommand("rm -rf " + s.folder())
	}
	if len(s.ServerName()) > 0 {
		defer s.runCommand("rm /srv/nginx/" + s.ServerName())
		defer s.runCommand("sudo certbot delete --cert-name " + s.ServerName())
	}

	if err := s.dropDatabase(); err != nil {
		return err
	}

	return nil
}

func (s *Server) runCommand(cmd string) error {
	fmt.Println("Executing " + cmd)
	c := exec.Command("sh", "-c", cmd)
	c.Dir = s.folder()
	output, err := c.Output()
	if err != nil {
		fmt.Println(fmt.Sprintf("Error: %s", err))
		return err
	}

	fmt.Println(s.folder())
	fmt.Println(cmd)
	fmt.Println(fmt.Sprintf("%s", output))

	return nil
}

func (s *Server) runCommands(cmds []string) error {
	for _, cmd := range cmds {
		if err := s.runCommand(cmd); err != nil {
			return err
		}
	}

	return nil
}
