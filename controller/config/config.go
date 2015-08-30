package config

type Config struct {
	Ip           string `json:"ip"`
	Port         int    `json:"port"`
	LogLevel     string `json:"log_level"`
	Syslog       bool   `json:"syslog"`
	*Tls         `json:"tls"`
	*DataStore   `json:"data_store"`
	*JobStore    `json:"job_store"`
	*DeviceStore `json:"device_store"`
}

type Tls struct {
	Enabled bool `json:"enabled"`
}

type DataStore struct {
	Type          string `json:"type"`
	*DataStoreGit `json:"git"`
}

type DataStoreGit struct {
	Url                    string `json:"url"`
	*DataStoreGitUser      `json:"user"`
	PoolSize               int      `json:"pool_size"`
	Structure              []string `json:"structure"`
	IncludeReadmeOverviews bool     `json:"include_readme_overviews"`
	DataDir                string   `json:"data_dir"`
}

type DataStoreGitUser struct {
	FriendlyName string `json:"friendly_name"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	Pass         string `json:"pass"`
}

type JobStore struct {
	Type           string `json:"type"`
	*JobStoreLocal `json:"local"`
}

type JobStoreLocal struct {
	JobGenerics map[string]*Job `json:"job_generics"`
	Jobs        map[string]*Job `json:"jobs"`
}

type Job struct {
	Generic      string `json:"generic"`
	*JobPrompt   `json:"prompt"`
	*JobSchedule `json:"schedule"`
	*JobCommand  `json:"command"`
}

type JobPrompt struct {
	EndsWith string `json:"string"`
}

type JobSchedule struct {
	Cron     string `json:"string"`
	Duration string `json:"duration"`
	Iso8601  string `json:"iso_8601"`
	Fixed    int64  `json:"fixed"`
}

type JobCommand struct {
	Inline []string `json:"inline"`
}

type DeviceStore struct {
	Type              string `json:"type"`
	*DeviceStoreLocal `json:"local"`
}

type DeviceStoreLocal struct {
	DeviceGenerics map[string]*Device `json:"device_generics"`
	Devices        map[string]*Device `json:"devices"`
}

type Device struct {
	Host               string `json:"protocol"`
	*DeviceProtocol    `json:"protocol"`
	Tags               []string `json:"tags"`
	*DeviceCredentials `json:"credentials"`
	Jobs               map[string]*Job `json:"jobs"`
}

type DeviceProtocol struct {
	Type               string `json:"type"`
	*DeviceProtocolSsh `json:"ssh"`
}

type DeviceProtocolSsh struct {
	Port int `json:"port"`
}

type DeviceCredentials struct {
	User   string `json:"user"`
	Pass   string `json:"pass"`
	Prompt string `json:"prompt"`
}
