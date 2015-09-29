package model

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/config"
)

type Job struct {
	Name        string `json:"name"`
	*CommandSet `json:"command_set"`
	FileSet     map[string]*JobFile `json:"file_set"`
	Schedule    `json:"-"`
}

func NewDefaultJob(name string) *Job {
	return &Job{Name: name}
}

func (j *Job) ApplyConfig(conf *config.Job) error {
	switch conf.Type {
	case "", "command":
		if conf.JobCommand != nil {
			if cmdSet, err := NewCommandSetFromConfig(conf.JobCommand); err != nil {
				return fmt.Errorf("Invalid command set: %v", err)
			} else {
				j.CommandSet = cmdSet
			}
		}
	case "file":
		if len(conf.JobFile) == 0 {
			return errors.New("At least one file required")
		}
		j.FileSet = make(map[string]*JobFile)
		for fileName, fileConf := range conf.JobFile {
			if fileName == "" {
				return errors.New("Empty filename")
			}
			if fileConf.Compression != "" && fileConf.Compression != "gzip" {
				return fmt.Errorf("Invalid compression '%v' for file", fileConf.Compression)
			}
			j.FileSet[fileName] = &JobFile{Compression: fileConf.Compression}
		}
	default:
		return fmt.Errorf("Unrecognized jov type %v", conf.Type)
	}
	if conf.JobSchedule != nil {
		if sched, err := NewScheduleFromConfig(conf.JobSchedule); err != nil {
			return fmt.Errorf("Invalid schedule: %v", err)
		} else {
			j.Schedule = sched
		}
	}
	return nil
}

func (j *Job) DeepCopy() *Job {
	// github.com/mitchellh/copystructure was failing because it could not traverse the pointer
	// so we have to do this ourselves.
	// TODO: write unit tests to confirm functionality doesn't change
	job := &Job{
		Name:     j.Name,
		Schedule: j.Schedule.DeepCopy(),
	}
	if j.CommandSet != nil {
		job.CommandSet = j.CommandSet.DeepCopy()
	}
	if j.FileSet != nil {
		job.FileSet = make(map[string]*JobFile)
		for k, v := range j.FileSet {
			job.FileSet[k] = v.DeepCopy()
		}
	}
	return j
}

func (j *Job) Validate() []error {
	// TODO: There is nothing else to validate right now
	return nil
}

type JobFile struct {
	Compression string `json:"compression"`
}

func (j *JobFile) DeepCopy() *JobFile {
	return &JobFile{Compression: j.Compression}
}
