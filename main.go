package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/vektorprogrammet/build-system/cli"
	"github.com/vektorprogrammet/build-system/handlers"

	"os"
)

func main() {
	keepRunning := cli.HandleArguments()
	if !keepRunning {
		return
	}

	secret := os.Getenv("GITHUB_WEBHOOKS_SECRET")
	webhookHandler := handlers.WebhookHandler{
		Secret: []byte(secret),
		Router: mux.NewRouter().PathPrefix("/webhooks/").Subrouter(),
	}

	webhookHandler.StartGitHubEventListeners()
	webhookHandler.InitRoutes()

	api := handlers.Api{
		Router: mux.NewRouter().PathPrefix("/api/").Subrouter(),
	}
	api.InitRoutes()

	serveMux := http.NewServeMux()

	serveMux.Handle("/webhooks/", webhookHandler.Router)
	serveMux.Handle("/api/", api.Router)

	handler := cors.Default().Handler(serveMux)

	fmt.Println("Listening to webhooks on port 5555")
	log.Fatal(http.ListenAndServe(":5555", handler))
}
