package worker

import (
	"encoding/json"
	"fmt"
	"gitlab.com/cretz/fusty/model"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

type Worker struct {
	conf                      *Config
	executionChan             chan *model.Execution
	runningExecutionCountLock *sync.Mutex
	runningExecutionCount     int
	errLog                    *log.Logger
	outLog                    *log.Logger
	tickLock                  *sync.Mutex
	controllerClient          *http.Client // TODO: give reasonable timeout and what not
}

func (w *Worker) RunWorker() error {
	panic("TODO")
}

func (w *Worker) NewWorker() error {
	panic("TODO")
}

func (w *Worker) Start() error {
	panic("TODO")
}

func (w *Worker) tick() {
	w.tickLock.Lock()
	defer w.tickLock.Unlock()
	// Make sure we're not full
	w.runningExecutionCountLock.Lock()
	jobsNeeded := w.conf.MaxJobs - w.runningExecutionCount
	w.runningExecutionCountLock.Unlock()
	if jobsNeeded <= 0 {
		return
	}
	// Ask for as many jobs as we need
	executions, err := w.nextExecutions(jobsNeeded)
	if err != nil {
		// We log and move on
		w.errLog.Printf("Unable to fetch next set of executions: %v", err)
		return
	}
	// Push em all to the channel
	for _, execution := range executions {
		w.executionChan <- execution
	}
}

func (w *Worker) nextExecutions(jobsNeeded int) ([]*model.Execution, error) {
	queryValues := url.Values{}
	for _, tag := range w.conf.Tags {
		queryValues.Add("tag", tag)
	}
	queryValues.Set("seconds", strconv.Itoa(w.conf.SleepSeconds))
	queryValues.Set("max", strconv.Itoa(jobsNeeded))
	url, err := url.Parse(w.conf.ControllerUrl + "/worker/next")
	if err != nil {
		// This is panic worthy
		panic(fmt.Errorf("Unable to parse URL: %v", err))
	}
	url.RawQuery = queryValues.Encode()
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		// This is panic worthy
		panic(fmt.Errorf("Unable to parse URL: %v", err))
	}
	req.Header.Set("Accept", "application/json")
	resp, err := w.controllerClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to obtain next set of jobs: %v", err)
	}
	if resp.StatusCode == 204 {
		return nil, nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Unable to read body: %v", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to retrieve next jobs, status %v, body: %v", resp.Status, string(body))
	}
	executions := make([]*model.Execution, 0)
	if err := json.Unmarshal(body, executions); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal JSON: %v. Body: %v", err, string(body))
	}
	return executions, nil
}
