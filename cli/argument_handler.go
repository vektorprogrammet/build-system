package cli

import "os"

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
	return true
}

func isFlag(arg string) bool {
	flags := []string{"-d", "--delete"}
	for i := 0; i < len(flags); i++ {
		if arg == flags[i] {
			return true
		}
	}
	return false
}