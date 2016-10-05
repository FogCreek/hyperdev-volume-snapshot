package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/snapshot/{projectId}", snapshotVolume).Methods("GET")
	http.ListenAndServe(":4444", router)
}

func snapshotVolume(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(req)
	projectId := vars["projectId"]

	log.Println("Snapshotting volume of project:", projectId)

	cmd := exec.Command("/bin/bash", "-c", SCRIPT, projectId)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		errMsg := fmt.Sprintf("%s: %s", err.Error(), stderr.String())
		log.Println(errMsg)
		http.Error(res, errMsg, http.StatusInternalServerError)
		return
	}

	outgoingJSON, err := json.Marshal(stdout.String())

	if err != nil {
		log.Println(err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(res, string(outgoingJSON))
}

var SCRIPT = `
#!/usr/bin/env bash
set -e

srcId=$1

if [ ! ${srcId} ]
then
    echo "Usage: rbd-snapshot source-volume"
    exit 1
fi

if [ ! $(rbd ls | grep -x ${srcId}) ]
then
    echo "Source volume ${srcId} does not exist"
    exit 1
fi

snapId=${srcId}@$(date -u -Iseconds)

fsfreeze -f /var/lib/docker-volumes/rbd/rbd/${srcId}

rbd snap create ${snapId}
rbd snap protect ${snapId}

fsfreeze -u /var/lib/docker-volumes/rbd/rbd/${srcId}

echo ${snapId}
`
