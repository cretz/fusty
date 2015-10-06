package model

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/config"
)

type Job struct {
	Name        string `json:"name"`
	*CommandSet `json:"command_set"`
	*FileSet    `json:"file_set"`
	Schedule    `json:"-"`
}

func NewDefaultJob(name string) *Job {
	return &Job{Name: name}
}

func (j *Job) ApplyConfig(conf *config.Job) error {
	switch conf.Type {
	case "", "command":
		if len(conf.Commands) > 0 {
			j.CommandSet = NewDefaultCommandSet()
			j.CommandSet.ApplyConfig(conf)
		}
	case "file":
		if len(conf.JobFile) == 0 {
			return errors.New("At least one file required")
		}
		j.FileSet = NewDefaultFileSet()
		j.FileSet.ApplyConfig(conf)
	default:
		return fmt.Errorf("Unrecognized job type %v", conf.Type)
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
		job.FileSet = j.FileSet.DeepCopy()
	}
	return job
}

func (j *Job) Validate() []error {
	errs := []error{}
	if j.Schedule == nil {
		errs = append(errs, errors.New("Job schedule required"))
	}
	if j.CommandSet != nil {
		errs = append(errs, j.CommandSet.Validate()...)
	}
	if j.FileSet != nil {
		errs = append(errs, j.FileSet.Validate()...)
	}
	return nil
}
