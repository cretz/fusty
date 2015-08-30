package controller

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/config"
	"gitlab.com/cretz/fusty/model"
)

type JobStore interface {
	AllJobs() map[string]*model.Job
}

func NewJobStoreFromConfig(conf *config.JobStore) (JobStore, error) {
	switch conf.Type {
	case "local":
		return newLocalJobStore(conf.JobStoreLocal)
	default:
		return nil, fmt.Errorf("Unrecognized job store type: %v", conf.Type)
	}
}

type localJobStore struct {
	jobs map[string]*model.Job
}

func newLocalJobStore(conf *config.JobStoreLocal) (*localJobStore, error) {
	store := &localJobStore{jobs: make(map[string]*model.Job)}
	errs := []error{}
	for name, confJob := range conf.Jobs {
		job := model.NewDefaultJob(name)
		// Generic first if present
		if confJob.Generic != "" {
			generic := conf.JobGenerics[confJob.Generic]
			if generic == nil {
				errs = append(errs, fmt.Errorf("Unable to find job generic named: %v", confJob.Generic))
				continue
			}
			if err := job.ApplyConfig(generic); err != nil {
				errs = append(errs, fmt.Errorf("Error applying job generic %v: %v", confJob.Generic, err))
				continue
			}
		} else if generic := conf.Jobs["default"]; generic != nil {
			if err := job.ApplyConfig(generic); err != nil {
				errs = append(errs, fmt.Errorf("Error applying default job generic: %v", err))
				continue
			}
		}
		// Specific job settings
		if err := job.ApplyConfig(confJob); err != nil {
			errs = append(errs, fmt.Errorf("Error configuring job %v: %v", job.Name, err))
			continue
		}
		// Validate the device
		if _, ok := store.jobs[job.Name]; ok {
			errs = append(errs, fmt.Errorf("Ambiguous job name %v", job.Name))
			continue
		}
		if validationErrors := job.Validate(); len(validationErrors) > 0 {
			errs = append(errs, validationErrors...)
			continue
		}
		store.jobs[job.Name] = job
	}
	// Any errors, combine into single error
	if len(errs) > 0 {
		msg := "Job validation failed:"
		for _, err := range errs {
			msg += "\n" + err.Error()
		}
		return nil, errors.New(msg)
	}
	return store, nil
}

func (l *localJobStore) AllJobs() map[string]*model.Job {
	// We trust callers not to modify this
	return l.jobs
}
