package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type GitHubEventMonitor struct {
	webhookSecretKey []byte
}

func (s *GitHubEventMonitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, s.webhookSecretKey)
	if err != nil {
		log.Fatal(err)
	}
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Fatal(err)
	}
	switch event := event.(type) {
	case *github.CommitCommentEvent:
		fmt.Println("CommitCommentEvent", event)
	case *github.CreateEvent:
		fmt.Println("CreateEvent", event)
	}
}

func githubstuff() {
	ctx := context.Background()
	accessToken := os.Getenv("GITHUB_ACCESS_TOKEN")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	opt := &github.PullRequestListOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	}

	PRs, _, err := client.PullRequests.List(ctx, "vektorprogrammet", "vektorprogrammet", opt)
	if err != nil {
		log.Fatal(err)
	}

	for _, pr := range PRs {
		fmt.Println(*pr.Title)
		fmt.Println(pr.Comments)
	}
}

func main() {
	githubstuff()
}
