package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"os"

	"github.com/google/go-github/github"
	"github.com/vektorprogrammet/build-system/staging"
	"golang.org/x/oauth2"
)

var eventChan chan interface{}

type GitHubEventMonitor struct {
	secret []byte
}

func (s *GitHubEventMonitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, s.secret)
	if err != nil {
		fmt.Printf("Failed to valdidate payload: %s\n", err)
		return
	}
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		fmt.Printf("Failed to parse webhook: %s\n", err)
		return
	}
	go func(event interface{}) {
		eventChan <- event
	}(event)
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

func startGitHubEventListener() {
	eventChan = make(chan interface{})
	go func() {
		for event := range eventChan {
			handler := WebhookHandler{}
			go handler.HandleEvent(event)
		}
	}()
}

func main() {
	startGitHubEventListener()
	secret := os.Getenv("GITHUB_WEBHOOKS_SECRET")
	eventMonitor := GitHubEventMonitor{secret: []byte(secret)}
	http.HandleFunc("/webhooks", eventMonitor.ServeHTTP)
	fmt.Println("Listening to webhooks on port 5555")
	log.Fatal(http.ListenAndServe(":5555", nil))
}

type WebhookHandler struct {}

func (handler WebhookHandler) HandleEvent(event interface{}) {
	e, ok := event.(*github.PullRequestEvent)
	if !ok {
		fmt.Println("Not a pull request event")
		return
	}

	if *e.Action != "opened" && *e.Action != "synchronize" {
		return
	}

	server := staging.Server{
		Branch: *e.PullRequest.Head.Ref,
		Repo: *e.Repo.URL,
		Domain: "staging.vektorprogrammet.no",
		RootFolder: "/var/www",
	}

	server.Deploy()
}
