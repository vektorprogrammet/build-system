package cli

import (
	"fmt"
	"github.com/vektorprogrammet/build-system/staging"
	"os/exec"
	"strings"
)

func ListServers() []staging.Server {
	dirs, err := ListDirContents(staging.GetDefaultRootFolder())
	if err != nil {
		fmt.Println(err)
		return nil
	}
	var servers []staging.Server
	for i := 0; i < len(dirs); i++ {
		server := staging.NewServer(dirs[i], nil)
		if server.Exists() {
			servers = append(servers, server)
		}
	}
	return servers
}

func ListDirContents(dir string) ([]string, error) {
	c := exec.Command("sh", "-c", "ls")
	c.Dir = dir
	output, err := c.Output()
	if err != nil {
		return nil, err
	}
	contents := strings.Split(string(output), "\n")
	return contents[:len(contents)-1], nil
}
