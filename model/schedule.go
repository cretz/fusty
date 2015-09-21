package model

import (
	"errors"
	"github.com/gorhill/cronexpr"
	"gitlab.com/cretz/fusty/config"
	"time"
)

type Schedule interface {
	// Exclusive
	Next(start time.Time) time.Time
	DeepCopy() Schedule
}

func NewScheduleFromConfig(sched *config.JobSchedule) (Schedule, error) {
	if sched.Cron == "" {
		return nil, errors.New("Only cron supported currently")
	}
	return NewCronSchedule(sched.Cron)
}

type CronSchedule struct {
	originalString string
	expr           *cronexpr.Expression
}

func NewCronSchedule(cron string) (*CronSchedule, error) {
	if expr, err := cronexpr.Parse(cron); err != nil {
		return nil, err
	} else {
		return &CronSchedule{originalString: cron, expr: expr}, nil
	}
}

func (c *CronSchedule) Next(start time.Time) time.Time {
	return c.expr.Next(start)
}

func (c *CronSchedule) DeepCopy() Schedule {
	ret, err := NewCronSchedule(c.originalString)
	if err != nil {
		panic(err)
	}
	return ret
}
