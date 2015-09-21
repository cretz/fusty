package model

import (
	"fmt"
	"gitlab.com/cretz/fusty/config"
)

type Job struct {
	Name        string `json:"name"`
	*CommandSet `json:"command_set"`
	Schedule    `json:"-"`
}

func NewDefaultJob(name string) *Job {
	return &Job{Name: name}
}

func (j *Job) ApplyConfig(conf *config.Job) error {
	if conf.JobCommand != nil {
		if cmdSet, err := NewCommandSetFromConfig(conf.JobCommand); err != nil {
			return fmt.Errorf("Invalid command set: %v", err)
		} else {
			j.CommandSet = cmdSet
		}
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
	return &Job{
		Name:       j.Name,
		CommandSet: j.CommandSet.DeepCopy(),
		Schedule:   j.Schedule.DeepCopy(),
	}
}

func (j *Job) Validate() []error {
	// TODO: There is nothing else to validate right now
	return nil
}
