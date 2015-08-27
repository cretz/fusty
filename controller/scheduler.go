package controller

import (
	"time"
)

type Scheduler interface {
	NextExecution(tags []string, before time.Time) *Execution
}

type Execution struct {
	Device    *Device   `json:"device"`
	Job       *Job      `json:"job"`
	Timestamp time.Time `json:"timestamp"`
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
	return ret, nil
}

func (j *schedulerLocal) NextExecution(tags []string, before time.Time) *Execution {
	// TODO: Should we balance our requests across tags better to prevent
	// multi-tag workers from getting all of one or another?
	// TODO: Also, we make no effort to get the very next one, we just grab a chunk
	if len(tags) {
		tags = []string{""}
	}
	for _, tag := range tags {
		for _, devJob := range j.deviceJobsByTag[tag] {
			if runTime := devJob.nextRun(before); !runTime.IsZero() {
				return &Execution{
					Device:    devJob.Device,
					Job:       devJob.Job,
					Timestamp: runTime,
				}
			}
		}
	}
	return nil
}

func (j *schedulerLocal) addDeviceJob(dev *Device) error {
	for _, job := range dev.Jobs {
		dev := &deviceJob{
			Device:  dev,
			Job:     job,
			lastRun: time.Now(),
		}
		if len(dev.Tags) == 0 {
			j.addDeviceJobToTag("", dev)
		} else {
			for _, tag := range dev.Tags {
				j.addDeviceJobToTag(tag, dev)
			}
		}
	}
	return nil
}

func (j *schedulerLocal) addDeviceJobToTag(tag string, d *deviceJob) {
	j.deviceJobsByTag[tag] = append(j.deviceJobsByTag[tag], d)
}

type deviceJob struct {
	*Device
	*Job
	lastRun time.Time
}

func (d deviceJob) nextRun(before time.Time) time.Time {
	ret := d.Job.LatestBetween(d.lastRun, before)
	if !ret.IsZero() {
		d.lastRun = ret
	}
	return ret
}
