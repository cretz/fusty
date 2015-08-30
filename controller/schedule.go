package controller

import (
	"errors"
	"github.com/gorhill/cronexpr"
	"gitlab.com/cretz/fusty/controller/config"
	"time"
)

type Schedule interface {
	// Exclusive
	LatestBetween(start time.Time, end time.Time) time.Time
}

func ScheduleFromConfig(sched *config.JobSchedule) (Schedule, error) {
	if sched.Cron == "" {
		return nil, errors.New("Only cron supported currently")
	}
	return NewCronSchedule(sched.Cron)
}

type CronSchedule struct {
	expr *cronexpr.Expression
}

func NewCronSchedule(cron string) (*CronSchedule, error) {
	if expr, err := cronexpr.Parse(cron); err != nil {
		return nil, err
	} else {
		return &CronSchedule{expr: expr}, nil
	}
}

func (c *CronSchedule) LatestBetween(start time.Time, end time.Time) time.Time {
	previous := time.Time{}
	for {
		res := c.expr.Next(start)
		if res.IsZero() || !res.Before(end) {
			return previous
		}
		previous = res
	}
}
