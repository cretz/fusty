package model

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/config"
)

type Job struct {
	Name string
	*CommandSet
	Schedule
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

func (d *Job) DeepCopy() *Job {
	panic("TODO")
}

func (d *Job) Validate() []error {
	return []error{errors.New("Not implemented")}
}
