package worker

import (
	"bytes"
	"fmt"
	"gitlab.com/cretz/fusty/model"
	"time"
)

type result struct {
	jobName        string
	deviceName     string
	jobTimestamp   int64
	startTimestamp int64
	endTimestamp   int64
	file           []byte // This can be nil/empty
	failure        error
}

func runExecution(execution *model.Execution) *result {
	res := &result{
		jobName:        execution.Job.Name,
		deviceName:     execution.Device.Name,
		jobTimestamp:   execution.Timestamp,
		startTimestamp: time.Now().Unix(),
	}
	sess, err := newSession(execution.Device)
	if err != nil {
		res.endTimestamp = time.Now().Unix()
		res.failure = fmt.Errorf("Unable to initiate session - %v", err)
		return res
	}
	defer sess.close()
	if err := sess.authenticate(execution.Device); err != nil {
		res.endTimestamp = time.Now().Unix()
		res.failure = fmt.Errorf("Authentication failed - %v", err)
		return res
	}
	res.file, res.failure = runJob(sess, execution.Job)
	res.endTimestamp = time.Now().Unix()
	return res
}

func runJob(sess session, job *model.Job) ([]byte, error) {
	// TODO: support ignoring errors?
	var buf bytes.Buffer
	// Run each command one at a time, failing on first error, otherwise concatenating
	// command results into one large buffer
	var err error
	for _, command := range job.CommandSet.Commands {
		bytes, commandErr := sess.run(command)
		if len(bytes) > 0 {
			if _, writeErr := buf.Write(bytes); writeErr != nil {
				// We know this will get overridden by command err if present which is a good thing
				err = fmt.Errorf("Unable to write to buffer: %v", writeErr)
			}
		}
		if commandErr != nil {
			err = fmt.Errorf("Command failed: %v", commandErr)
		}
		if err != nil {
			break
		}
	}
	return buf.Bytes(), err
}
