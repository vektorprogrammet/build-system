package staging

import (
	"fmt"

	"bytes"

	"io/ioutil"
	"os"

	"golang.org/x/crypto/ssh"
)

type environmentError struct {
	s string
}

func (e *environmentError) Error() string {
	return "Environment error: " + e.s
}

func UploadNginxConfig(configContent, serverName string) error {
	createNginxConfig := fmt.Sprintf("echo '%s' > /srv/nginx/%s", configContent, serverName)

	_, err := remoteRun(createNginxConfig)
	if err != nil {
		return err
	}

	_, err = remoteRun("sudo service nginx restart")
	return err
}

func SecureDomainWithSsl(domain string) error {
	_, err := remoteRun(fmt.Sprintf("printf '2\n' | sudo certbot --nginx -d %s", domain))
	return err
}

func CopyBaseWebsiteTo(targetDir string) error {
	cacheDir := "/var/www/" + targetDir + "/app/cache"
	logsDir := "/var/www/" + targetDir + "/app/logs"
	imageDir := "/var/www/" + targetDir + "/www/images"
	signatureDir := "/var/www/" + targetDir + "/signatures"

	copyFiles := fmt.Sprintf("cp -R /var/www/vektorprogrammet '/var/www/%s'", targetDir)
	_, err := remoteRun(copyFiles)
	if err != nil {
		return err
	}

	updateWebServerPermissions := fmt.Sprintf("setfacl -R -m u:www-data:rwX %s %s %s %s", cacheDir, logsDir, imageDir, signatureDir)
	_, err = remoteRun(updateWebServerPermissions)
	return err

	updateDefaultWebServerPermissions := fmt.Sprintf("setfacl -dR -m u:www-data:rwX %s %s %s %s", cacheDir, logsDir, imageDir, signatureDir)
	_, err = remoteRun(updateDefaultWebServerPermissions)
	return err
}

func remoteRun(cmd string) (string, error) {
	user := os.Getenv("VEKTOR_STAGING_USER")
	if len(user) == 0 {
		return "", &environmentError{s: "VEKTOR_STAGING_USER not set"}
	}

	addr := os.Getenv("VEKTOR_STAGING_ADDR")
	if len(user) == 0 {
		return "", &environmentError{s: "VEKTOR_STAGING_ADDR not set"}
	}

	privateKey := os.Getenv("PRIVATE_KEY")
	if len(user) == 0 {
		return "", &environmentError{s: "PRIVATE_KEY not set"}
	}

	fp, err := os.Open(privateKey)
	if err != nil {
		return "", err
	}
	defer fp.Close()

	buf, _ := ioutil.ReadAll(fp)
	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return "", err
	}

	// Authentication
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //TODO: Replace with fixed host key
	}
	// Connect
	client, err := ssh.Dial("tcp", addr+":22", config)
	if err != nil {
		return "", err
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b // get output

	err = session.Run(cmd)
	return b.String(), err
}
