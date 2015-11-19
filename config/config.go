package config

type Config struct {
	Ip           string `json:"ip,omitempty" toml:"ip" yaml:"ip,omitempty" hcl:"ip"`
	Port         int    `json:"port,omitempty" toml:"port" yaml:"port,omitempty" hcl:"port"`
	LogLevel     string `json:"log_level,omitempty" toml:"log_level" yaml:"log_level,omitempty" hcl:"log_level"`
	Syslog       bool   `json:"syslog,omitempty" toml:"syslog" yaml:"syslog,omitempty" hcl:"syslog"`
	MaxJobBytes  int64  `json:"max_job_bytes,omitempty" toml:"max_job_bytes" yaml:"max_job_bytes,omitempty" hcl:"max_job_bytes"`
	*Tls         `json:"tls,omitempty" toml:"tls" yaml:"tls,omitempty" hcl:"tls" hcl:"tls"`
	*DataStore   `json:"data_store,omitempty" toml:"data_store" yaml:"data_store,omitempty" hcl:"data_store"`
	*JobStore    `json:"job_store,omitempty" toml:"job_store" yaml:"job_store,omitempty" hcl:"job_store"`
	*DeviceStore `json:"device_store,omitempty" toml:"device_store" yaml:"device_store,omitempty" hcl:"device_store"`
}

type Tls struct {
	CertFile string `json:"cert_file,omitempty" toml:"cert_file" yaml:"cert_file,omitempty" hcl:"cert_file"`
	KeyFile  string `json:"key_file,omitempty" toml:"key_file" yaml:"key_file,omitempty" hcl:"key_file"`
}

type DataStore struct {
	Type          string `json:"type,omitempty" toml:"type" yaml:"type,omitempty" hcl:"type"`
	*DataStoreGit `json:"git,omitempty" toml:"git" yaml:"git,omitempty" hcl:"git"`
}

type DataStoreGit struct {
	Url                    string `json:"url,omitempty" toml:"url" yaml:"url" hcl:"url"`
	*DataStoreGitUser      `json:"user,omitempty" toml:"user" yaml:"user,omitempty" hcl:"user"`
	PoolSize               int      `json:"pool_size,omitempty" toml:"pool_size" yaml:"pool_size,omitempty" hcl:"pool_size"`
	Structure              []string `json:"structure,omitempty" toml:"structure" yaml:"structure,omitempty" hcl:"structure"`
	IncludeReadmeOverviews bool     `json:"include_readme_overviews,omitempty" toml:"include_readme_overviews" yaml:"include_readme_overviews,omitempty" hcl:"include_readme_overviews"`
	DataDir                string   `json:"data_dir,omitempty" toml:"data_dir" yaml:"data_dir,omitempty" hcl:"data_dir"`
}

type DataStoreGitUser struct {
	FriendlyName string `json:"friendly_name,omitempty" toml:"friendly_name" yaml:"friendly_name,omitempty" hcl:"friendly_name"`
	Email        string `json:"email,omitempty" toml:"email" yaml:"email,omitempty" hcl:"email"`
	Name         string `json:"name,omitempty" toml:"name" yaml:"name,omitempty" hcl:"name"`
	Pass         string `json:"pass,omitempty" toml:"pass" yaml:"pass,omitempty" hcl:"pass"`
}

type JobStore struct {
	Type           string `json:"type,omitempty" toml:"type" yaml:"type,omitempty" hcl:"type"`
	*JobStoreLocal `json:"local,omitempty" toml:"local" yaml:"local,omitempty" hcl:"local"`
}

type JobStoreLocal struct {
	JobGenerics map[string]*Job `json:"job_generics,omitempty" toml:"job_generics" yaml:"job_generics,omitempty" hcl:"job_generics"`
	Jobs        map[string]*Job `json:"jobs,omitempty" toml:"jobs" yaml:"jobs,omitempty" hcl:"jobs"`
}

type Job struct {
	Generic        string `json:"generic,omitempty" toml:"generic" yaml:"generic,omitempty" hcl:"generic"`
	Type           string `json:"type,omitempty" toml:"type" yaml:"type,omitempty" hcl:"type"`
	*JobSchedule   `json:"schedule,omitempty" toml:"schedule" yaml:"schedule,omitempty" hcl:"schedule"`
	Commands       []*JobCommand       `json:"commands,omitempty" toml:"commands" yaml:"commands,omitempty" hcl:"commands"`
	CommandGeneric *JobCommand         `json:"command_generic,omitempty" toml:"command_generic" yaml:"command_generic,omitempty" hcl:"command_generic"`
	JobFile        map[string]*JobFile `json:"file,omitempty" toml:"file" yaml:"file,omitempty" hcl:"file"`
	Scrubbers      []*JobScrubber      `json:"scrubbers,omitempty" toml:"scrubbers" yaml:"scrubbers,omitempty" hcl:"scrubbers"`
	TemplateValues map[string]string   `json:"template_values,omitempty" toml:"template_values" yaml:"template_values,omitempty" hcl:"template_values"`
}

type JobSchedule struct {
	Cron     string `json:"cron,omitempty" toml:"cron" yaml:"cron,omitempty" hcl:"cron"`
	Duration string `json:"duration,omitempty" toml:"duration" yaml:"duration,omitempty" hcl:"duration"`
	Iso8601  string `json:"iso_8601,omitempty" toml:"iso_8601" yaml:"iso_8601,omitempty" hcl:"iso_8601"`
	Fixed    int64  `json:"fixed,omitempty" toml:"fixed" yaml:"fixed,omitempty" hcl:"fixed"`
}

type JobCommand struct {
	Command       string   `json:"command,omitempty" toml:"command" yaml:"command,omitempty" hcl:"command"`
	Expect        []string `json:"expect,omitempty" toml:"expect" yaml:"expect,omitempty" hcl:"expect"`
	ExpectNot     []string `json:"expect_not,omitempty" toml:"expect_not" yaml:"expect_not,omitempty" hcl:"expect_not"`
	Timeout       *int     `json:"timeout,omitempty" toml:"timeout" yaml:"timeout,omitempty" hcl:"timeout"`
	ImplicitEnter *bool    `json:"implicit_enter,omitempty" toml:"implicit_enter" yaml:"implicit_enter,omitempty" hcl:"implicit_enter"`
}

type JobFile struct {
	Compression string `json:"compression,omitempty" toml:"compression" yaml:"compression,omitempty" hcl:"compression"`
}

type JobScrubber struct {
	Type    string `json:"type,omitempty" toml:"type" yaml:"type,omitempty" hcl:"type"`
	Search  string `json:"search,omitempty" toml:"search" yaml:"search,omitempty" hcl:"search"`
	Replace string `json:"replace,omitempty" toml:"replace" yaml:"replace,omitempty" hcl:"replace"`
}

type DeviceStore struct {
	Type              string `json:"type,omitempty" toml:"type" yaml:"type,omitempty" hcl:"type"`
	*DeviceStoreLocal `json:"local,omitempty" toml:"local" yaml:"local,omitempty" hcl:"local"`
}

type DeviceStoreLocal struct {
	DeviceGenerics map[string]*Device `json:"device_generics,omitempty" toml:"device_generics" yaml:"device_generics,omitempty" hcl:"device_generics"`
	Devices        map[string]*Device `json:"devices,omitempty" toml:"devices" yaml:"devices,omitempty" hcl:"devices"`
}

type Device struct {
	Generic            string `json:"generic,omitempty" toml:"generic" yaml:"generic,omitempty" hcl:"generic"`
	Host               string `json:"host,omitempty" toml:"host" yaml:"host,omitempty" hcl:"host"`
	*DeviceProtocol    `json:"protocol,omitempty" toml:"protocol" yaml:"protocol,omitempty" hcl:"protocol"`
	Tags               []string `json:"tags,omitempty" toml:"tags" yaml:"tags,omitempty" hcl:"tags"`
	*DeviceCredentials `json:"credentials,omitempty" toml:"credentials" yaml:"credentials,omitempty" hcl:"credentials"`
	Jobs               map[string]*Job `json:"jobs,omitempty" toml:"jobs" yaml:"jobs,omitempty" hcl:"jobs"`
}

type DeviceProtocol struct {
	Type               string `json:"type,omitempty" toml:"type" yaml:"type,omitempty" hcl:"type"`
	*DeviceProtocolSsh `json:"ssh,omitempty" toml:"ssh" yaml:"ssh,omitempty" hcl:"ssh"`
}

type DeviceProtocolSsh struct {
	Port              int  `json:"port,omitempty" toml:"port" yaml:"port,omitempty" hcl:"port"`
	IncludeCbcCiphers bool `json:"include_cbc_ciphers,omitempty" toml:"include_cbc_ciphers" yaml:"include_cbc_ciphers,omitempty" hcl:"include_cbc_ciphers"`
}

type DeviceCredentials struct {
	User string `json:"user,omitempty" toml:"user" yaml:"user,omitempty" hcl:"user"`
	Pass string `json:"pass,omitempty" toml:"pass" yaml:"pass,omitempty" hcl:"pass"`
}
