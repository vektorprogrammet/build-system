package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/vektorprogrammet/build-system/messenger"
	"github.com/vektorprogrammet/build-system/staging"
)

var eventChan chan interface{}

type WebhookHandler struct{
	Secret []byte
	Router *mux.Router
}

func (wh *WebhookHandler) InitRoutes() {
	wh.Router.HandleFunc("/github", wh.handleWebhook)
	wh.startGitHubEventListeners()
}

func (wh *WebhookHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, wh.Secret)
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

func (wh *WebhookHandler) startGitHubEventListeners() {
	eventChan = make(chan interface{})
	go func() {
		for event := range eventChan {
			wh.handlePullRequestEvent(event)
			wh.handleBranchDeleteEvent(event)
			wh.handlePushEvent(event)
		}
	}()
}

func (wh *WebhookHandler) handlePushEvent(event interface{}) {
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

func (wh *WebhookHandler) handleBranchDeleteEvent(event interface{}) {
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

func (wh WebhookHandler) handlePullRequestEvent(event interface{}) {
	e, ok := event.(*github.PullRequestEvent)
	if !ok {
		fmt.Println("Not a pull request event")
		return
	}

	if !(*e.Action == "opened" || *e.Action == "synchronize" || *e.Action == "reopened") {
		return
	}

	commenter := messenger.GithubCommenter{
		PrNumber: *e.PullRequest.Number,
	}
	server := staging.NewServer(*e.PullRequest.Head.Ref, commenter.UpdateProgress)

	if server.Exists() {
		if server.CanBeFastForwarded() {
			server.Update()
		}
		fmt.Println("Staging server updated at https://" + server.ServerName())
	} else {
		commenter.StartingDeploy()
		err := server.Deploy()
		if err != nil {
			fmt.Printf("Could not create staging server: %s\n", err)
			commenter.EditComment(commenter.ProgressCommentId, "Could not deploy staging server because of an error")
			server.Remove()
		} else {
			commenter.EditComment(commenter.ProgressCommentId, "Staging server deployed at https://"+server.ServerName())
		}
	}
}