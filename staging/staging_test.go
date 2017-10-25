package staging

import (
	"testing"

	"github.com/kristandre/BuildSystem/nginx"
)

var config = nginx.Config{
	ServerName:        "abcd.staging.vektorprogrammet.no",
	Root:              "/var/www/abcd.staging.vektorprogrammet.no/www",
}

func TestDeployStagingInstance(t *testing.T) {
	serverName := config.ServerName

	CopyBaseWebsiteTo(serverName)
	UploadNginxConfig(config.String(), serverName)
	SecureDomainWithSsl(serverName)
}
