package staging

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/vektorprogrammet/build-system/nginx"
)

type Server struct {
	Repo           string
	Branch         string
	RootFolder     string
	Domain         string
	UpdateProgress func(message string, progress int)
}

const DefaultRepo = "https://github.com/vektorprogrammet/vektorprogrammet"

const DefaultRootFolder = "/var/www/servers"

const DefaultDomain = "staging.vektorprogrammet.no"

const DefaultInstallationFolder = "/var/www/staging-server"

func NewServer(branch string, updateProgress func(message string, progress int)) Server {
	s := Server{}
	// Default values
	s.Repo = DefaultRepo
	s.RootFolder = DefaultRootFolder
	s.Domain = DefaultDomain

	// Initialize fields
	s.Branch = branch
	s.UpdateProgress = updateProgress

	return s
}

func (s *Server) MarshalJSON() ([]byte, error) {
	var tmp struct {
		Repo   string `json:"repo"`
		Branch string `json:"branch"`
		Domain string `json:"domain"`
		Url    string `json:"url"`
	}
	tmp.Repo = s.Repo
	tmp.Branch = s.Branch
	tmp.Domain = s.Domain
	tmp.Url = "https://" + s.ServerName()

	return json.Marshal(&tmp)
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

	if err := s.createSetupParametersFile(); err != nil {
		return err
	}

	if err := s.createRobotsTxt(); err != nil {
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

	if err := s.createParametersFile(); err != nil {
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

func (s *Server) CanBeFastForwarded() bool {
	c := exec.Command("sh", "-c", "git remote update")
	c.Dir = s.folder()
	_, err := c.Output()
	if err != nil {
		fmt.Printf("Could not update remotes: %s\n", err)
		return false
	}

	c = exec.Command("sh", "-c", "git status")
	c.Dir = s.folder()
	output, err := c.Output()
	if err != nil {
		fmt.Printf("Could not execute git status: %s\n", err)
		return false
	}
	return strings.Contains(string(output), "can be fast-forwarded") ||
		strings.Contains(string(output), "have diverged")
}

func (s *Server) Update() error {
	if err := s.runCommand(fmt.Sprintf("git reset --hard origin/%s", s.Branch)); err != nil {
		return err
	}

	if err := s.runCommand(fmt.Sprintf("git pull origin %s", s.Branch)); err != nil {
		return err
	}

	if err := s.createRobotsTxt(); err != nil {
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
	varDir := s.folder() + "/var"
	webDir := s.folder() + "/web"

	return s.runCommands([]string{
		"setfacl -R -m u:vektorprogrammet:rwX .",
		"setfacl -dR -m u:vektorprogrammet:rwX .",
		fmt.Sprintf("setfacl -R -m u:www-data:rwX %s %s", varDir, webDir),
		fmt.Sprintf("setfacl -dR -m u:www-data:rwX %s %s", varDir, webDir),
	})
}

func (s *Server) createNginxConfig() error {
	nginxConfig := nginx.Config{
		Root:       s.folder() + "/web",
		ServerName: s.ServerName(),
	}

	return s.runCommand(fmt.Sprintf("echo '%s' > /srv/nginx/%s", nginxConfig.String(), nginxConfig.ServerName))
}

func (s *Server) createRobotsTxt() error {
	robotContent := "User-agent: *\nDisallow: /"
	return s.runCommand(fmt.Sprintf("echo '%s' > %s/web/robots.txt", robotContent, s.folder()))
}

func (s *Server) restartNginx() error {
	return s.runCommand(fmt.Sprintf("sudo service nginx restart"))
}

func (s *Server) secureWithHttps() error {
	return s.runCommand(fmt.Sprintf("printf '2\n' | sudo certbot --nginx -d %s", s.ServerName()))
}

func (s *Server) folder() string {
	return s.RootFolder + "/" + s.safeBranch()
}

func (s *Server) safeBranch() string {
	b := s.Branch
	b = strings.Replace(b, "/", "", -1)
	b = strings.Replace(b, "_", "-", -1)
	b = strings.ToLower(b)
	return b
}
func (s *Server) ServerName() string {
	return s.safeBranch() + "." + s.Domain
}

func (s *Server) install() error {
	var wg sync.WaitGroup
	wg.Add(3)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		s.runCommand("php ./composer.phar install -n --no-dev --optimize-autoloader")
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
			"NODE_ENV=staging npm run setup:client",
			"NODE_ENV=staging npm run build:client",
		})
	}(&wg)

	wg.Wait()

	return nil
}

func (s *Server) createDatabase() error {
	commands := []string{
		"php bin/console doctrine:database:create",
		"php bin/console doctrine:schema:create",
		"php bin/console doctrine:fixtures:load -n",
		"php bin/console doctrine:migrations:version --add --all -n",
	}

	return s.runCommands(commands)
}

func (s *Server) updateDatabase() error {

	return s.runCommand("php bin/console doctrine:migrations:migrate -n ")
}

func (s *Server) createSetupParametersFile() error {
	cmd := fmt.Sprintf("cp %s/parameters_during_setup.yml app/config/parameters.yml", DefaultInstallationFolder)
	if err := s.runCommand(cmd); err != nil {
		return err
	}

	return s.runCommand(fmt.Sprintf("sed -i 's/dbname/%s/g' app/config/parameters.yml", s.safeBranch()))
}

func (s *Server) createParametersFile() error {
	cmd := fmt.Sprintf("cp %s/parameters.yml app/config/parameters.yml", DefaultInstallationFolder)
	if err := s.runCommand(cmd); err != nil {
		return err
	}

	err := s.runCommand(fmt.Sprintf("sed -i 's/dbname/%s/g' app/config/parameters.yml", s.safeBranch()))
	if err != nil {
		return err
	}

	return s.runCommand("php ./composer.phar install -n --no-dev --optimize-autoloader")
}

func (s *Server) dropDatabase() error {
	return s.runCommand("php bin/console doctrine:database:drop --force")
}

func (s *Server) Remove() error {
	if len(s.folder()) > len(s.RootFolder)+1 {
		defer s.runCommand("rm -rf " + s.folder())
	}
	if len(s.ServerName()) > 0 {
		defer s.runCommands([]string{"rm /srv/nginx/" + s.ServerName(), "sudo service nginx restart"})
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
