package model

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/config"
	"regexp"
	"strings"
)

type Job struct {
	Name           string `json:"name"`
	*CommandSet    `json:"command_set"`
	*FileSet       `json:"file_set"`
	Schedule       `json:"-"`
	Scrubbers      []*JobScrubber    `json:"scrubbers"`
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
	if conf.JobSchedule != nil {
		if sched, err := NewScheduleFromConfig(conf.JobSchedule); err != nil {
			return fmt.Errorf("Invalid schedule: %v", err)
		} else {
			j.Schedule = sched
		}
	}
	for _, scrubber := range conf.Scrubbers {
		j.Scrubbers = append(j.Scrubbers, NewJobScrubberFromConfig(scrubber))
	}
	for key, value := range conf.TemplateValues {
		j.TemplateValues[key] = value
	}
	return nil
}

func (j *Job) ApplyTemplateValues() {
	for key, value := range j.TemplateValues {
		if j.CommandSet != nil {
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
		for _, scrubber := range j.Scrubbers {
			scrubber.Search = strings.Replace(scrubber.Search, "{{"+key+"}}", value, -1)
			scrubber.Replace = strings.Replace(scrubber.Replace, "{{"+key+"}}", value, -1)
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
	for _, scrubber := range j.Scrubbers {
		job.Scrubbers = append(job.Scrubbers, scrubber.DeepCopy())
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
	for _, scrubber := range j.Scrubbers {
		if err := scrubber.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("Scrubbe validation failed: %v", err))
		}
	}
	return errs
}

type JobScrubber struct {
	Type    string `json:"type"`
	Search  string `json:"search"`
	Replace string `json:"replace"`
}

func NewJobScrubberFromConfig(conf *config.JobScrubber) *JobScrubber {
	// We don't need to eagerly check the type, we'll check in Validate
	typ := conf.Type
	if typ == "" {
		typ = "simple"
	}
	search := conf.Search
	if typ == "regex" || typ == "regex_substitute" {
		search = sanitizeRegex(search)
	}
	return &JobScrubber{
		Type:    typ,
		Search:  search,
		Replace: conf.Replace,
	}
}

func (j *JobScrubber) DeepCopy() *JobScrubber {
	return &JobScrubber{
		Type:    j.Type,
		Search:  j.Search,
		Replace: j.Replace,
	}
}

func (j *JobScrubber) Validate() error {
	if j.Search == "" {
		return errors.New("No search given")
	}
	if j.Type == "regex" || j.Type == "regex_substitute" {
		// We only validate when there are no replacers because replacers
		// can completely change regex semantics
		if !strings.Contains(j.Search, "{{") {
			if _, err := regexp.Compile(j.Search); err != nil {
				return fmt.Errorf("Invalid regex '%v': %v", j.Search, err)
			}
		}
	} else if j.Type != "simple" {
		return fmt.Errorf("Unrecognized scrubber type: %v", j.Type)
	}
	return nil
}
