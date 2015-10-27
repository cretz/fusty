package model

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/config"
	"strings"
)

type Job struct {
	Name           string `json:"name"`
	*CommandSet    `json:"command_set"`
	*FileSet       `json:"file_set"`
	Schedule       `json:"-"`
	TemplateValues map[string]string `json:"template_values"`
}

func NewDefaultJob(name string) *Job {
	return &Job{Name: name, TemplateValues: map[string]string{}}
}

func (j *Job) ApplyConfig(conf *config.Job) error {
	// The type could already be determined by generic...
	if conf.Type == "command" {
		if j.FileSet != nil {
			return errors.New("Generic file set in job of type command")
		}
		if j.CommandSet == nil {
			j.CommandSet = NewDefaultCommandSet()
		}
	} else if conf.Type == "file" {
		if j.CommandSet != nil {
			return errors.New("Generic command set in job of type file")
		}
		if j.FileSet == nil {
			j.FileSet = NewDefaultFileSet()
		}
	} else if conf.Type == "" {
		// Default to command set
		if j.FileSet == nil && j.CommandSet == nil {
			j.CommandSet = NewDefaultCommandSet()
		}
	} else {
		return fmt.Errorf("Unrecognized job type %v", conf.Type)
	}
	// Now we can apply config as necessary
	if j.CommandSet != nil {
		j.CommandSet.ApplyConfig(conf)
	}
	if j.FileSet != nil {
		j.FileSet.ApplyConfig(conf)
	}
	//	switch conf.Type {
	//	case "", "command":
	//		if j.CommandSet == nil {
	//			j.CommandSet = NewDefaultCommandSet()
	//		}
	//		j.CommandSet.ApplyConfig(conf)
	//	case "file":
	//		if j.FileSet == nil {
	//			j.FileSet = NewDefaultFileSet()
	//		}
	//		j.FileSet.ApplyConfig(conf)
	//	default:
	//		return fmt.Errorf("Unrecognized job type %v", conf.Type)
	//	}
	if conf.JobSchedule != nil {
		if sched, err := NewScheduleFromConfig(conf.JobSchedule); err != nil {
			return fmt.Errorf("Invalid schedule: %v", err)
		} else {
			j.Schedule = sched
		}
	}
	for key, value := range conf.TemplateValues {
		j.TemplateValues[key] = value
	}
	return nil
}

func (j *Job) ApplyTemplateValues() {
	if j.CommandSet != nil {
		for key, value := range j.TemplateValues {
			for _, cmd := range j.CommandSet.Commands {
				cmd.Command = strings.Replace(cmd.Command, "{{"+key+"}}", value, -1)
				for index, expect := range cmd.Expect {
					cmd.Expect[index] = strings.Replace(expect, "{{"+key+"}}", value, -1)
				}
				for index, expect := range cmd.ExpectNot {
					cmd.ExpectNot[index] = strings.Replace(expect, "{{"+key+"}}", value, -1)
				}
			}
		}
	}
}

func (j *Job) DeepCopy() *Job {
	// github.com/mitchellh/copystructure was failing because it could not traverse the pointer
	// so we have to do this ourselves.
	// TODO: write unit tests to confirm functionality doesn't change
	job := &Job{
		Name:           j.Name,
		Schedule:       j.Schedule.DeepCopy(),
		TemplateValues: map[string]string{},
	}
	if j.CommandSet != nil {
		job.CommandSet = j.CommandSet.DeepCopy()
	}
	if j.FileSet != nil {
		job.FileSet = j.FileSet.DeepCopy()
	}
	for key, value := range j.TemplateValues {
		job.TemplateValues[key] = value
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
	return errs
}
