package worker

import "time"

type Config struct {
	// sans trailing slash
	ControllerUrl string
	Tags          []string
	SleepSeconds  int
	MaxJobs       int
	Timeout       time.Duration
}
