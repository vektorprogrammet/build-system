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

type githubCommenter struct {
	accessToken       string
	progressCommentId int64
	prNumber          int
}

func (g *githubCommenter) createClient() (*github.Client, context.Context) {
	ctx := context.Background()
	accessToken := os.Getenv("GITHUB_ACCESS_TOKEN")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc), ctx
}

func (g *githubCommenter) comment(comment string) (*github.IssueComment, error) {
	client, ctx := g.createClient()

	prComment := github.IssueComment{
		Body: &comment,
	}

	issue, _, err := client.Issues.CreateComment(ctx, "vektorprogrammet", "vektorprogrammet", g.prNumber, &prComment)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

func (g *githubCommenter) editComment(id int64, comment string) (*github.IssueComment, error) {
	client, ctx := g.createClient()

	prComment := github.IssueComment{
		Body: &comment,
	}

	issue, _, err := client.Issues.EditComment(ctx, "vektorprogrammet", "vektorprogrammet", id, &prComment)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

func (g *githubCommenter) StartingDeploy() {
	issueComment, _ := g.comment("Starting deploy to staging server...")
	g.progressCommentId = *issueComment.ID
}

func (g *githubCommenter) UpdateProgress(message string, progress int) {
	comment := fmt.Sprintf("Deploying this pull request to the staging server... %d %%\n%s", progress, message)
	g.editComment(g.progressCommentId, comment)
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
	fmt.Println("Secret: " + secret)
	eventMonitor := GitHubEventMonitor{secret: []byte(secret)}

	http.HandleFunc("/webhooks", eventMonitor.ServeHTTP)
	fmt.Println("Listening to webhooks on port 5555")
	log.Fatal(http.ListenAndServe(":5555", nil))
}

type WebhookHandler struct{}

func (handler WebhookHandler) HandleEvent(event interface{}) {
	e, ok := event.(*github.PullRequestEvent)
	if !ok {
		fmt.Println("Not a pull request event")
		return
	}

	if *e.Action != "opened" && *e.Action != "synchronize" {
		return
	}

	commenter := githubCommenter{
		prNumber: *e.PullRequest.Number,
	}
	server := staging.Server{
		Branch:         *e.PullRequest.Head.Ref,
		Repo:           *e.Repo.CloneURL,
		Domain:         "staging.vektorprogrammet.no",
		RootFolder:     "/var/www",
		UpdateProgress: commenter.UpdateProgress,
	}

	if server.Exists() {
		server.Update()
		commenter.comment("Staging server updated at https://" + server.ServerName())
	} else {
		commenter.StartingDeploy()
		server.Deploy()
		commenter.editComment(commenter.progressCommentId, "Staging server deployed at https://"+server.ServerName())
	}
}
