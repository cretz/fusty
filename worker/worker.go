package worker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/go-syslog"
	"gitlab.com/cretz/fusty/model"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Worker struct {
	conf                      *Config
	runningExecutionCountLock *sync.Mutex
	runningExecutionCount     int
	errLog                    *log.Logger
	outLog                    *log.Logger
	tickLock                  *sync.Mutex
	controllerClient          *http.Client
}

// configFileName can be empty which means default config
func RunWorker(conf *Config) error {
	// Our responsibility to validate config here
	if conf.ControllerUrl == "" {
		return errors.New("Controller URL required")
	} else if _, err := url.Parse(conf.ControllerUrl); err != nil {
		return fmt.Errorf("Invalid controller URL %v: %v", conf.ControllerUrl, err)
	}
	conf.ControllerUrl = strings.TrimRight(conf.ControllerUrl, "/")

	// We need to ping the controller to make sure it's good
	if resp, err := http.Get(conf.ControllerUrl + "/worker/ping"); err != nil {
		return fmt.Errorf("Unable to contact controller at %v/worker/ping: %v", conf.ControllerUrl, err)
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bad status from %v/worker/ping: %v", conf.ControllerUrl, resp.StatusCode)
	}

	// Create the worker and go
	work, err := NewWorker(conf)
	if err != nil {
		return fmt.Errorf("Unable to start worker: %v", err)
	}
	work.Start()
	return nil
}

func NewWorker(conf *Config) (*Worker, error) {
	worker := &Worker{
		conf: conf,
		runningExecutionCountLock: &sync.Mutex{},
		tickLock:                  &sync.Mutex{},
	}

	// Setup loggers
	if conf.Syslog {
		if logger, err := gsyslog.NewLogger(gsyslog.LOG_ERR, "LOCAL0", "fusty"); err != nil {
			return nil, fmt.Errorf("Unable to create syslog: %v", err)
		} else {
			worker.errLog = log.New(logger, "", log.LstdFlags)
		}
		if logger, err := gsyslog.NewLogger(gsyslog.LOG_INFO, "LOCAL0", "fusty"); err != nil {
			return nil, fmt.Errorf("Unable to create syslog: %v", err)
		} else {
			worker.outLog = log.New(logger, "", log.LstdFlags)
		}
	} else {
		worker.errLog = log.New(os.Stderr, "", log.LstdFlags)
		worker.outLog = log.New(os.Stdout, "", log.LstdFlags)
	}

	// Setup HTTP client with timeout and disallow redirects
	worker.controllerClient = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return errors.New("Redirects disabled") },
		Timeout:       time.Duration(conf.TimeoutSeconds) * time.Second,
	}
	return worker, nil
}

// This blocks and never ends except in a panic
func (w *Worker) Start() {
	for {
		w.tick()
		// We will sleep half the amount of time range configured
		// to fetch jobs for.
		time.Sleep(time.Duration(w.conf.SleepSeconds/2) * time.Second)
	}
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
	// Schedule em all
	for _, execution := range executions {
		w.scheduleExecution(execution)
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

func (w *Worker) scheduleExecution(execution *model.Execution) {
	t := time.Unix(execution.Timestamp, 0)
	// We don't even hold the resulting timer because we don't care
	time.AfterFunc(t.Sub(time.Now()), func() { w.runExecutionAndPostResult(execution) })
}

func (w *Worker) runExecutionAndPostResult(execution *model.Execution) {
	// Increment running job count
	w.runningExecutionCountLock.Lock()
	w.runningExecutionCount += 1
	w.runningExecutionCountLock.Unlock()

	// Run job
	result := runExecution(execution)

	// Decrement running job count
	w.runningExecutionCountLock.Lock()
	w.runningExecutionCount -= 1
	w.runningExecutionCountLock.Unlock()

	// Post response to controller
	url, err := url.Parse(w.conf.ControllerUrl + "/worker/complete")
	if err != nil {
		// This is panic worthy
		panic(fmt.Errorf("Unable to parse URL: %v", err))
	}
	body := &bytes.Buffer{}
	var postFailedErr error
	formWriter := multipart.NewWriter(body)
	postFailedErr = formWriter.WriteField("job", result.jobName)
	if postFailedErr == nil {
		postFailedErr = formWriter.WriteField("device", result.deviceName)
	}
	if postFailedErr == nil {
		postFailedErr = formWriter.WriteField("job_timestamp", strconv.FormatInt(result.jobTimestamp, 10))
	}
	if postFailedErr == nil {
		postFailedErr = formWriter.WriteField("start_timestamp", strconv.FormatInt(result.startTimestamp, 10))
	}
	if postFailedErr == nil {
		postFailedErr = formWriter.WriteField("end_timestamp", strconv.FormatInt(result.endTimestamp, 10))
	}
	if postFailedErr == nil && result.failure != nil {
		postFailedErr = formWriter.WriteField("failure", result.failure.Error())
	}
	if postFailedErr == nil && len(result.file) > 0 {
		if file, err := formWriter.CreateFormFile("file", result.jobName); err != nil {
			postFailedErr = fmt.Errorf("Unable to create form file HTTP param: %v", err)
		} else if _, err := file.Write(result.file); err != nil {
			postFailedErr = fmt.Errorf("Unable to write bytes to HTTP param: %v", err)
		}
	}
	if postFailedErr == nil {
		postFailedErr = formWriter.Close()
	} else {
		formWriter.Close()
	}
	if postFailedErr == nil {
		req, err := http.NewRequest("POST", url.String(), body)
		postFailedErr = err
		if err == nil {
			req.Header.Set("Content-Type", formWriter.FormDataContentType())
			resp, err := w.controllerClient.Do(req)
			postFailedErr = err
			if err != nil {
				defer resp.Body.Close()
				body, err := ioutil.ReadAll(resp.Body)
				postFailedErr = err
				if err == nil && resp.StatusCode != 200 {
					postFailedErr = fmt.Errorf(
						"Controller failed to accept post with status %v, body: %v", resp.Status, string(body))
				}
			}
		}
	}
	// TODO: provide facilities to rotate or at least debounce these logs in case tens of thousands happen
	if postFailedErr != nil {
		w.errLog.Printf("Error sending controller completed job %v on device %v that started on %v: %v",
			result.jobName, result.deviceName, time.Unix(result.startTimestamp, 0), postFailedErr)
	}
}
