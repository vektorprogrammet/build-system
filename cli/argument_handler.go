package cli

import (
	"fmt"
	"os"
)

func HandleArguments() (keepRunning bool) {
	if len(os.Args) > 2 && os.Args[1] == "deploy-branch" {
		if len(os.Args) > 3 && (os.Args[2] == "-d" || os.Args[2] == "--delete") {
			err := StopServer(os.Args[3])
			if err != nil {
				println(err)
			}
			return false
		} else {
			err := DeployBranch(os.Args[2])
			if err != nil {
				println(err)
			}
			return false
		}
	}

	if len(os.Args) == 2 && (os.Args[1] == "list-servers" || os.Args[1] == "ls") {
		servers := ListServers()
		for i := 0; i < len(servers); i++ {
			fmt.Printf("%s ", servers[i].Branch)
		}
		fmt.Printf("\n")
	}

	if len(os.Args) > 1 {
		fmt.Printf("Unrecognized command %s\n", os.Args[1])
		return false
	} else {
		return true
	}
}
