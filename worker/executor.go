package worker

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/model"
	"io"
	"io/ioutil"
	"log"
	"regexp"
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
	if job.FileSet != nil {
		return fetchFile(sess, job)
	} else if job.CommandSet != nil {
		return runCommands(sess, job)
	} else {
		return nil, errors.New("Unable to find file set or command set to run")
	}
}

func runCommands(sess session, job *model.Job) ([]byte, error) {
	shell, err := sess.shell()
	if err != nil {
		return nil, fmt.Errorf("Unable to open shell: %v", err)
	}
	defer shell.close()
	buff := []byte{}
	shellReader := bufio.NewReader(shell)
	for _, cmd := range job.CommandSet.Commands {
		if _, err := shell.Write([]byte(cmd.Command)); err != nil {
			return buff, fmt.Errorf("Error writing command '%v': %v", cmd.Command, err)
		}
		if cmd.ImplicitEnter {
			if _, err := shell.Write([]byte{13}); err != nil {
				return buff, fmt.Errorf("Error entering after command '%v': %v", cmd.Command, err)
			}
		}
		// Due to how we don't store job state from job to job, we recompile the regex
		// every command here knowing it is not too expensive in most cases. Here if the
		// timeout is not zero we check the output once a second.
		if cmd.Timeout == 0 {
			continue
		}
		expectRegex := []*regexp.Regexp{}
		expectNotRegex := []*regexp.Regexp{}
		for _, exp := range cmd.Expect {
			if expr, err := regexp.Compile(exp); err != nil {
				return buff, fmt.Errorf("Unable to compile regex '%v': %v", exp, err)
			} else {
				expectRegex = append(expectRegex, expr)
			}
		}
		for _, exp := range cmd.ExpectNot {
			if expr, err := regexp.Compile(exp); err != nil {
				return buff, fmt.Errorf("Unable to compile regex '%v': %v", exp, err)
			} else {
				expectNotRegex = append(expectNotRegex, expr)
			}
		}

		matchSuccess := false
		thisBytes := []byte{}
		for i := 0; i < cmd.Timeout; i++ {
			time.Sleep(time.Second)
			// Fetch all output since our last go
			for {
				n, err := shellReader.Read(thisBytes[len(thisBytes):])
				if n > 0 {
					// Write what we read
					buff = append(buff, thisBytes[len(thisBytes)-n:]...)
				}
				if err != nil && err != io.EOF {
					return buff, fmt.Errorf("Failure reading output of command '%v': %v", cmd.Command, err)
				}
				if n == 0 {
					break
				}
			}
			// Check for failure expectations
			for i, notExpr := range expectNotRegex {
				if notExpr.Match(thisBytes) {
					return buff, fmt.Errorf("Output of command '%v' matched failure pattern: %v",
						cmd.Command, cmd.ExpectNot[i])
				}
			}
			// Now for success
			for _, expr := range expectRegex {
				if expr.Match(thisBytes) {
					matchSuccess = true
					break
				}
			}
		}
		if len(expectRegex) > 0 && !matchSuccess {
			return buff, fmt.Errorf("Output of command '%v' never matched expected pattern(s)", cmd.Command)
		}
	}
	return buff, nil
}

func fetchFile(sess session, job *model.Job) ([]byte, error) {
	// Just sftp files for now
	// Get all the paths and sort in alphabetical order
	paths := []string{}
	pathsToFiles := map[string]*model.FileSetFile{}
	for _, file := range job.FileSet.Files {
		paths = append(paths, file.Name)
		pathsToFiles[file.Name] = file
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
		if pathsToFiles[path].Compression == "gzip" {
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
