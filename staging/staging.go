package staging

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/vektorprogrammet/build-system/nginx"
)

type Server struct {
	Repo string
	Branch string
	RootFolder string
	Domain string
}

func (s *Server) Deploy() error {
	if err := s.createServerFolder(); err != nil {
		return err
	}

	if err := s.clone(); err != nil {
		return err
	}

	if err := s.checkout(); err != nil {
		return err
	}

	if err := s.install(); err != nil {
		return err
	}

	if err := s.setFolderPermissions(); err != nil {
		return err
	}

	if err := s.createNginxConfig(); err != nil {
		return err
	}

	if err := s.secureWithHttps(); err != nil {
		return err
	}

	return nil
}

func (s *Server) Update() error {
	if err := s.clone(); err != nil {
		return err
	}

	return s.setFolderPermissions()
}

func (s *Server) createServerFolder() error {
	return s.runCommand("mkdir -p " + s.folder())
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

	updateWebServerPermissions := fmt.Sprintf("setfacl -R -m u:www-data:rwX %s %s %s %s", cacheDir, logsDir, imageDir, signatureDir)
	err := s.runCommand(updateWebServerPermissions)
	if err != nil {
		return err
	}

	updateDefaultWebServerPermissions := fmt.Sprintf("setfacl -dR -m u:www-data:rwX %s %s %s %s", cacheDir, logsDir, imageDir, signatureDir)
	return s.runCommand(updateDefaultWebServerPermissions)
}

func (s *Server) createNginxConfig() error {
	nginxConfig := nginx.Config{
		Root: s.folder() + "/www",
		ServerName: s.serverName(),
	}

	return s.runCommand(fmt.Sprintf("echo '%s' > /srv/nginx/%s", nginxConfig.String(), nginxConfig.ServerName))
}

func (s *Server) restartNginx() error {
	return s.runCommand(fmt.Sprintf("sudo service nginx restart"))
}

func (s *Server) secureWithHttps() error {
	return s.runCommand(fmt.Sprintf("printf '2\n' | sudo certbot --nginx -d %s", s.serverName()))
}

func (s *Server) folder() string {
	return s.RootFolder + "/" + s.Branch
}

func (s *Server) serverName() string {
	return strings.Replace(s.Branch, "_", "-", -1) + "." + s.Domain
}

func (s *Server) install() error {
	commands := []string{
		"SYMFONY_ENV=prod php ./composer.phar install -n --no-dev --optimize-autoloader",
		"npm install",
		"npm run build:prod",
		"npm run setup:scheduling",
		"npm run build:scheduling",
		"php app/console doctrine:database:create --env=prod",
		"php app/console doctrine:schema:create --env=prod",
		"php app/console doctrine:fixtures:load -n",
		"php app/console cache:clear --env=prod",
	}

	return s.runCommands(commands)
}

func (s *Server) createParametersFile() error {
	if err := s.runCommand("cp ../parameters.yml app/config/parameters.yml"); err != nil{
		return err
	}

	return s.runCommand(fmt.Sprintf("sed -i 's/dbname/%s/g' app/config/parameters.yml", s.Branch))
}

func (s *Server) dropDatabase() error {
	return s.runCommand("php app/console doctrine:database:drop --force")
}

func (s *Server) remove() error {
	defer s.runCommand("rm -rf " + s.folder())

	if err := s.dropDatabase(); err != nil{
		return err
	}

	return nil
}

func (s *Server) runCommand(cmd string) error {
	c := exec.Command("sh", "-c", cmd)
	c.Dir = s.folder()
	output, err := c.Output()
	if err != nil {
		return err
	}

	fmt.Println(s.folder())
	fmt.Println(cmd)
	fmt.Println(fmt.Sprintf("%s", output))

	return nil
}

func (s *Server) runCommands(cmds []string) error {
	for _, cmd := range cmds {
		c := exec.Command("sh", "-c", cmd)
		c.Dir = s.folder()
		output, err := c.Output()
		if err != nil {
			return err
		}

		fmt.Println(s.folder())
		fmt.Println(cmd)
		fmt.Println(fmt.Sprintf("%s", output))
	}

	return nil
}
