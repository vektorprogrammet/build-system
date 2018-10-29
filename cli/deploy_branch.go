package cli

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/vektorprogrammet/build-system/staging"
	"github.com/vektorprogrammet/build-system/messenger"
	"os"
)

func DeployBranch(branchName string) error {
	ctx := context.Background()
	client := github.NewClient(nil)
	slack := messenger.NewSlack(os.Getenv("SLACK_ENDPOINT"), "#staging_log", "vektorbot", ":robot_face:")

	if err := EnsureBranchExists(ctx, client, branchName); err != nil {
		return err
	}

	server := staging.NewServer(branchName, func(message string, progress int) {
		fmt.Printf("%s %d\n", message, progress)
		slack.Send(fmt.Sprintf("%s: %s %d %%", branchName, message, progress))
	})

	if server.Exists() {
		fmt.Println("Server exists. Forcing update...")
		slack.Send(fmt.Sprintf("%s: %s", branchName, "Server exists. Forcing update..."))
		if server.CanBeFastForwarded() {
			server.Update()
			fmt.Println("Server updated.")
			slack.Send(fmt.Sprintf("%s: %s", branchName, "Server updated."))
		} else {
			fmt.Println("Did not update: Branch is up to date with origin/master")
			slack.Send(fmt.Sprintf("%s: %s", branchName, "Did not update: Branch is up to date with origin/master"))
		}

	} else {
		err := server.Deploy()
		if err != nil {
			server.Remove()
			fmt.Printf("Could not create staging server: %s\n", err)
			slack.Send(fmt.Sprintf("%s: Could not create staging server: %s", branchName, err))
		} else {
			fmt.Printf("Staging server deployed at https://%s\n", server.ServerName())
			slack.Send(fmt.Sprintf("%s: Staging server deployed at https://%s", branchName, server.ServerName()))
		}
	}

	return nil
}

func StopServer(branchName string) error {
	ctx := context.Background()
	client := github.NewClient(nil)

	if err := EnsureBranchExists(ctx, client, branchName); err != nil {
		return err
	}

	server := staging.NewServer(branchName, func(message string, progress int) {
		fmt.Printf("%s %d\n", message, progress)
	})

	if server.Exists() {
		fmt.Printf("Stopping server hosting %s\n", branchName)
		err := server.Remove()
		if err != nil {
			fmt.Println("Could not remove branch")
			return err
		}
	} else {
		fmt.Printf("No staging server deployed for branch %s\n", branchName)
	}

	return nil
}

func EnsureBranchExists(ctx context.Context, client *github.Client, branchName string) error {
	_, _, err := client.Git.GetRef(ctx, "vektorprogrammet", "vektorprogrammet", "refs/heads/"+branchName)
	if err != nil {
		fmt.Printf("Could not find branch %s: %s\n", branchName, err)
		return err
	}
	return nil
}
