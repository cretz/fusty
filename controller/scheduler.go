package controller

import (
	"encoding/json"
	"gitlab.com/cretz/fusty/model"
	"log"
	"sync"
	"time"
)

type Scheduler interface {
	NextExecution(tags []string, before time.Time) *model.Execution
}

type schedulerLocal struct {
	deviceJobsByTag map[string][]*deviceJob
}

func (c *Controller) NewLocalScheduler() (Scheduler, error) {
	ret := &schedulerLocal{deviceJobsByTag: make(map[string][]*deviceJob)}
	for _, dev := range c.AllDevices() {
		if err := ret.addDeviceJob(dev); err != nil {
			return nil, err
		}
	}
	if Verbose {
		if text, err := json.MarshalIndent(ret.deviceJobsByTag, "", "  "); err == nil {
			log.Printf("Full device-job set by tag in scheduler:\n%v\n", string(text))
		}
	}
	return ret, nil
}

func (j *schedulerLocal) NextExecution(tags []string, before time.Time) *model.Execution {
	// TODO: Should we balance our requests across tags better to prevent
	// multi-tag workers from getting all of one or another?
	// TODO: Also, we make no effort to get the very next one, we just grab a chunk
	if len(tags) == 0 {
		tags = []string{""}
	}
	for _, tag := range tags {
		for _, devJob := range j.deviceJobsByTag[tag] {
			if runTime := devJob.nextRun(before); !runTime.IsZero() {
				return &model.Execution{
					Device:    devJob.Device,
					Job:       devJob.Job,
					Timestamp: runTime.Unix(),
				}
			}
		}
	}
	return nil
}

func (j *schedulerLocal) addDeviceJob(dev *model.Device) error {
	for _, job := range dev.Jobs {
		devJob := &deviceJob{
			Device:      dev,
			Job:         job,
			lastRun:     time.Now(),
			lastRunLock: &sync.Mutex{},
		}
		if len(devJob.Tags) == 0 {
			j.addDeviceJobToTag("", devJob)
		} else {
			for _, tag := range devJob.Tags {
				j.addDeviceJobToTag(tag, devJob)
			}
		}
		if Verbose {
			log.Printf("Added job %v for device %v which will likely run next at %v",
				devJob.Job.Name, devJob.Device.Name, devJob.Job.Next(devJob.lastRun))
		}
	}
	return nil
}

func (j *schedulerLocal) addDeviceJobToTag(tag string, d *deviceJob) {
	j.deviceJobsByTag[tag] = append(j.deviceJobsByTag[tag], d)
}

type deviceJob struct {
	*model.Device `json:"device"`
	*model.Job    `json:"job"`
	lastRun       time.Time   `json:"-"`
	lastRunLock   *sync.Mutex `json:"-"`
}

func (d *deviceJob) nextRun(before time.Time) time.Time {
	d.lastRunLock.Lock()
	defer d.lastRunLock.Unlock()
	after := time.Now()
	if d.lastRun.After(after) {
		after = d.lastRun
	}
	ret := d.Job.Next(after)
	if ret.After(after) && ret.Before(before) {
		d.lastRun = ret
		return ret
	}
	return time.Time{}
}
