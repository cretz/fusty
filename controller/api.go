package controller

import (
	"encoding/json"
	"fmt"
	"gitlab.com/cretz/fusty/model"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// 500 meg
const MaxJobBytes int64 = 524288000

func (c *Controller) addApiHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/worker/ping", c.apiWorkerPing)
	mux.HandleFunc("/worker/next", c.apiWorkerNext)
	mux.HandleFunc("/worker/complete", c.apiWorkerComplete)
}

func (c *Controller) apiWorkerPing(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
	} else {
		w.WriteHeader(http.StatusOK)
	}
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
	executions := make([]*model.Execution, 0, max)
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
		http.Error(w, fmt.Sprintf("Internal error: %v", err), http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}
}

func (c *Controller) apiWorkerComplete(w http.ResponseWriter, req *http.Request) {
	var maxBytes = MaxJobBytes
	if c.conf.MaxJobBytes != 0 {
		maxBytes = c.conf.MaxJobBytes
	}
	if err := req.ParseMultipartForm(maxBytes); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	// Try to read contents
	contents := []byte{}
	if files := req.MultipartForm.File["file"]; len(files) == 1 {
		f, err := files[0].Open()
		if err != nil {
			http.Error(w, "Unable to read file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		contents, err = ioutil.ReadAll(f)
		if err != nil {
			http.Error(w, "Unable to read file: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// Build job and validate
	job := &DataStoreJob{
		JobName:    singleMutlipartFormValOrEmpty("job", req),
		DeviceName: singleMutlipartFormValOrEmpty("device", req),
		JobTime:    timestampOrZero("job_timestamp", req),
		StartTime:  timestampOrZero("start_timestamp", req),
		EndTime:    timestampOrZero("end_timestamp", req),
		Failure:    singleMutlipartFormValOrEmpty("failure", req),
		Contents:   contents,
	}
	if job.JobName == "" || job.DeviceName == "" ||
		job.JobTime.IsZero() || job.StartTime.IsZero() || job.EndTime.IsZero() {
		http.Error(w,
			"Fields job, device, job_timestamp, start_timestamp, end_timestamp are required", http.StatusBadRequest)
		return
	} else if job.Failure == "" && len(job.Contents) == 0 {
		http.Error(w, "Failure and contents may not both be empty", http.StatusBadRequest)
		return
	}
	// TODO: do not store failures, log em instead
	c.DataStore.Store(job)
	w.WriteHeader(http.StatusOK)
}

func timestampOrZero(name string, req *http.Request) time.Time {
	if str := singleMutlipartFormValOrEmpty(name, req); str == "" {
		return time.Time{}
	} else if i, err := strconv.ParseInt(str, 10, 0); err != nil {
		return time.Time{}
	} else {
		return time.Unix(i, 0)
	}
}

func singleMutlipartFormValOrEmpty(name string, req *http.Request) string {
	vals := req.MultipartForm.Value[name]
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}
