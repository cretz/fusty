package worker

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/model"
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

	// We scrub no matter what but if there is failure we don't override failure.
	// Note, if there is anything to scrub we completely remove what exists on
	// failure because we don't want to send over unscrubbed info
	if len(res.file) > 0 && len(execution.Job.Scrubbers) > 0 {
		if clean, err := scrubBytes(res.file, execution.Job); err != nil {
			if res.failure == nil {
				res.failure = err
			}
			res.file = []byte{}
		} else {
			res.file = clean
		}
	}

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
	if Verbose {
		log.Printf("Connecting to shell to run job %v", job.Name)
	}
	shell, err := sess.shell()
	if err != nil {
		return nil, fmt.Errorf("Unable to open shell: %v", err)
	}
	defer shell.close()
	buff := []byte{}
	// Wait a second and clear output before first command
	// TODO: this makes things a bit slow :-( ...maybe some kind of thing that knows when it's at the first prompt
	time.Sleep(time.Second)
	shell.bytesAndReset()
	for _, cmd := range job.CommandSet.Commands {
		if Verbose {
			log.Printf("Running command '%v' for job %v", cmd.Command, job.Name)
		}
		// Clear out all pending output before running the command by reading everything in the buffer
		// (but still hold on to it)
		buff = append(buff, shell.bytesAndReset()...)
		// Write the command
		if _, err := shell.Write([]byte(cmd.Command)); err != nil {
			return buff, fmt.Errorf("Error writing command '%v': %v", cmd.Command, err)
		}
		if cmd.ImplicitEnter {
			if Verbose {
				log.Printf("Sending implicit enter for job %v", job.Name)
			}
			if _, err := shell.Write([]byte{10}); err != nil {
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
		thisCommandBytes := []byte{}
		if Verbose {
			log.Printf("Reading log output for command '%v'", cmd.Command)
		}
	CommandLoop:
		for i := 0; i < cmd.Timeout; i++ {
			currBytes := shell.bytesAndReset()
			thisCommandBytes = append(thisCommandBytes, currBytes...)
			buff = append(buff, currBytes...)
			if Verbose && len(thisCommandBytes) > 0 {
				log.Printf("Current output for command '%v':\n----\n%v\n----", cmd.Command, string(thisCommandBytes))
			}
			// Check for failure expectations
			for i, notExpr := range expectNotRegex {
				if notExpr.Match(thisCommandBytes) {
					if Verbose {
						log.Printf("Matched unexpected pattern %v", cmd.Expect[i])
					}
					return buff, fmt.Errorf("Output of command '%v' matched failure pattern: %v",
						cmd.Command, cmd.ExpectNot[i])
				}
			}
			// Now for success
			for i, expr := range expectRegex {
				if expr.Match(thisCommandBytes) {
					if Verbose {
						log.Printf("Matched expected pattern %v", cmd.Expect[i])
					}
					matchSuccess = true
					break CommandLoop
				}
			}
			// We go ahead and sleep the one second
			time.Sleep(time.Second)
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

func scrubBytes(dirty []byte, job *model.Job) ([]byte, error) {
	clean := dirty
	for _, scrubber := range job.Scrubbers {
		if Verbose {
			log.Printf("Scrubber type %v replacing '%v' with '%v'", scrubber.Type, scrubber.Search, scrubber.Replace)
		}
		if scrubber.Type == "regex" || scrubber.Type == "regex_substitute" {
			// Meh not that harmful to recompile here everytime
			if exp, err := regexp.Compile(scrubber.Search); err != nil {
				return nil, fmt.Errorf("Unable to compile scrubber search '%v': %v", scrubber.Search, err)
			} else if scrubber.Type == "regex" {
				clean = exp.ReplaceAllLiteral(clean, []byte(scrubber.Replace))
			} else {
				clean = exp.ReplaceAll(clean, []byte(scrubber.Replace))
			}
		} else {
			clean = bytes.Replace(clean, []byte(scrubber.Search), []byte(scrubber.Replace), -1)
		}
	}
	return clean, nil
}
