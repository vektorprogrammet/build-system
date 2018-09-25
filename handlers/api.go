package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vektorprogrammet/build-system/staging"
)

type Api struct {
	Router *mux.Router
}

func (a *Api) InitRoutes() {
	a.Router.HandleFunc("/servers", a.handleGetServers)
	a.Router.HandleFunc("/disk-space", a.handleGetDiskSpace)
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

func (a *Api) handleGetDiskSpace(w http.ResponseWriter, r *http.Request) {
	size, used, err := getDiskSpaceInfo(staging.DefaultRootFolder)
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	diskSpaceInfo := struct {
		Size int `json:"size"`
		Used int `json:"used"`
	}{
		size, used,
	}
	diskSpaceJson, err := json.Marshal(diskSpaceInfo)
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Println(diskSpaceJson)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(diskSpaceJson)
}

func getDiskSpaceInfo(dir string) (size int, used int, err error) {
	c := exec.Command("sh", "-c", "df | grep /dev/vda1")
	c.Dir = staging.DefaultRootFolder
	output, err := c.Output()
	if err != nil {
		return 0, 0, err
	}
	outputFields := strings.Fields(string(output[:]))
	size, err = strconv.Atoi(outputFields[1])
	if err != nil {
		return 0, 0, err
	}
	used, err = strconv.Atoi(outputFields[2])
	if err != nil {
		return 0, 0, err
	}
	fmt.Printf("space: %i %i\n", size, used)
	return size, used, err
}
