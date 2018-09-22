package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/vektorprogrammet/build-system/cli"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

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
		fmt.Printf("Failed to validate payload: %s\n", err)
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
		handler := WebhookHandler{}
		for event := range eventChan {
			handler.HandlePullRequestEvent(event)
			handler.HandleBranchDeleteEvent(event)
			handler.HandlePushEvent(event)
		}
	}()
}

func getServers(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(staging.DefaultRootFolder)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var servers []staging.Server
	for _, f := range files {
		if f.IsDir() {
			servers = append(servers, staging.NewServer(f.Name(), func(message string, progress int) {}))
		}
	}

	serversJson, err := json.Marshal(servers)
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(serversJson)
}

func main() {
	keepRunning := cli.HandleArguments()
	if !keepRunning {
		return
	}

	startGitHubEventListener()
	secret := os.Getenv("GITHUB_WEBHOOKS_SECRET")
	fmt.Println("Secret: " + secret)
	eventMonitor := GitHubEventMonitor{secret: []byte(secret)}

	http.HandleFunc("/webhooks", eventMonitor.ServeHTTP)
	http.HandleFunc("/api/servers", getServers)
	fmt.Println("Listening to webhooks on port 5555")
	log.Fatal(http.ListenAndServe(":5555", nil))
}

type WebhookHandler struct{}

func (handler WebhookHandler) HandlePushEvent(event interface{}) {
	e, ok := event.(*github.PushEvent)
	if !ok {
		fmt.Println("Not a push event")
		return
	}

	branch := strings.Split(e.GetRef(), "/")[2]
	server := staging.NewServer(branch, func(message string, progress int) {
		fmt.Printf("%s %d\n", message, progress)
	})

	if server.Exists() && server.CanBeFastForwarded() {
		server.Update()
		fmt.Printf("Staging server updated at https://" + server.ServerName())
	}
}

func (handler WebhookHandler) HandleBranchDeleteEvent(event interface{}) {
	e, ok := event.(*github.DeleteEvent)
	if !ok {
		fmt.Println("Not a delete event")
		return
	}

	if *e.RefType != "branch" {
		fmt.Print("Not a branch")
		return
	}

	server := staging.NewServer(*e.Ref, func(message string, progress int) {
		fmt.Printf("%s %d\n", message, progress)
	})

	if server.Exists() {
		err := server.Remove()
		if err != nil {
			fmt.Println("Could not remove branch")
		}
	}
}

func (handler WebhookHandler) HandlePullRequestEvent(event interface{}) {
	e, ok := event.(*github.PullRequestEvent)
	if !ok {
		fmt.Println("Not a pull request event")
		return
	}

	if !(*e.Action == "opened" || *e.Action == "synchronize" || *e.Action == "reopened") {
		return
	}

	commenter := githubCommenter{
		prNumber: *e.PullRequest.Number,
	}
	server := staging.NewServer(*e.PullRequest.Head.Ref, commenter.UpdateProgress)

	if server.Exists() {
		if server.CanBeFastForwarded() {
			server.Update()
		}
		commenter.comment("Staging server updated at https://" + server.ServerName())
	} else {
		commenter.StartingDeploy()
		err := server.Deploy()
		if err != nil {
			fmt.Printf("Could not create staging server: %s\n", err)
			commenter.editComment(commenter.progressCommentId, "Could not deploy staging server because of an error")
			server.Remove()
		} else {
			commenter.editComment(commenter.progressCommentId, "Staging server deployed at https://"+server.ServerName())
		}
	}
}
