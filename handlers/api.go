package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vektorprogrammet/build-system/staging"
)

type Api struct {
	Router *mux.Router
}

func (a *Api) InitRoutes() {
	a.Router.HandleFunc("/servers", a.handleGetServers)
}

func (a *Api) handleGetServers(w http.ResponseWriter, r *http.Request) {
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
