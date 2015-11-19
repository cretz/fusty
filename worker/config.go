package worker

type Config struct {
	// sans trailing slash
	ControllerUrl  string
	Tags           []string
	SleepSeconds   int
	MaxJobs        int
	TimeoutSeconds int
	Syslog         bool
	SkipVerify     bool
	CAFile         string
}
