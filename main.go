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

	cmd := exec.Command("/bin/bash", "-c", fmt.Sprintf(SCRIPT, projectId))
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

srcId=%s

if [ ! ${srcId} ]
then
    >&2 echo -n "Usage: rbd-snapshot source-volume"
    exit 1
fi

if [ ! -d "/var/lib/docker-volumes/rbd/rbd/${srcId}" ]
then
    >&2 echo -n "Source volume ${srcId} is not mounted"
    exit 1
fi

snapId=${srcId}_$( date +%%s%%N | cut -b1-13 )

fsfreeze -f /var/lib/docker-volumes/rbd/rbd/${srcId}

rbd cp ${srcId} ${snapId}

fsfreeze -u /var/lib/docker-volumes/rbd/rbd/${srcId}

echo -n ${snapId}
`
