package worker

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"gitlab.com/cretz/fusty/model"
	"io/ioutil"
	"log"
	"sort"
	"strings"
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

var (
	fileContentsHr string = strings.Repeat("-", 12)
)

func runExecution(execution *model.Execution) *result {
	res := &result{
		jobName:        execution.Job.Name,
		deviceName:     execution.Device.Name,
		jobTimestamp:   execution.Timestamp,
		startTimestamp: time.Now().Unix(),
	}
	if Verbose {
		log.Printf("Running execution: %v", res)
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
	// We don't support command yet...
	if job.CommandSet != nil {
		panic("Command sets not supported yet")
	}
	// Just sftp files for now
	// Get all the paths and sort in alphabetical order
	paths := []string{}
	for path, _ := range job.FileSet {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	// Run for each, decompressing as needed
	var buf bytes.Buffer
	for i, path := range paths {
		if Verbose {
			log.Printf("Fetching file: %v", path)
		}
		// Any error is an error for all
		fileBytes, err := sess.fetchFile(path)
		if err != nil {
			return nil, err
		}
		if job.FileSet[path].Compression == "gzip" {
			gzipReader, err := gzip.NewReader(bytes.NewReader(fileBytes))
			if err != nil {
				return nil, fmt.Errorf("Unable to begin decompressing file %v: %v", path, err)
			}
			defer gzipReader.Close()
			fileBytes, err = ioutil.ReadAll(gzipReader)
			if err != nil {
				return nil, fmt.Errorf("Unable to decompress file %v: %v", path, err)
			}
		}
		// If there are multiple files, we separate each section with the path
		if len(paths) > 1 {
			fileBytes = append([]byte(fileContentsHr+"\nFile: "+path+"\n"+fileContentsHr+"\n"), fileBytes...)
		}
		// Any one after the first must have a newline prepended
		if i > 0 {
			fileBytes = append([]byte("\n"), fileBytes...)
		}
		if _, err := buf.Write(fileBytes); err != nil {
			return nil, fmt.Errorf("Error writing contents to buffer: %v", err)
		}
	}
	if Verbose {
		log.Printf("Overall fetched:\n%v", string(buf.Bytes()))
	}
	return buf.Bytes(), nil
}
