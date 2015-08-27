package controller

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func (c *Controller) addApiHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/worker/next", c.apiWorkerNext)
	mux.HandleFunc("/worker/complete", c.apiWorkerNext)
}

func (c *Controller) apiWorkerNext(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	uri, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tags := uri.Query()["tag"]
	seconds := 15
	max := 15
	if secondsParam := uri.Query()["seconds"]; len(secondsParam) == 1 {
		if v, err := strconv.Atoi(secondsParam[0]); err != nil {
			http.Error(w, "Invalid seconds", http.StatusBadRequest)
			return
		} else {
			seconds = v
		}
	}
	if maxParam := uri.Query()["max"]; len(maxParam) == 1 {
		if v, err := strconv.Atoi(maxParam[0]); err != nil {
			http.Error(w, "Invalid max", http.StatusBadRequest)
			return
		} else {
			max = v
		}
	}
	executions := make([]*Execution, 0, max)
	fromNow := time.Now().Add(time.Duration(seconds) * time.Second)
	for i := 0; i < max; i++ {
		if execution := c.NextExecution(tags, fromNow); execution == nil {
			break
		} else {
			executions = append(executions, execution)
		}
	}
	if len(executions) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if body, err := json.Marshal(executions); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}
}

func (c *Controller) apiWorkerComplete(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("TODO\n"))
}
