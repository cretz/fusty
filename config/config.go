package config

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Ip           string `json:"ip,omitempty"`
	Port         int    `json:"port,omitempty"`
	LogLevel     string `json:"log_level,omitempty"`
	Syslog       bool   `json:"syslog,omitempty"`
	MaxJobBytes  int64  `json:"max_job_bytes,omitempty"`
	*Tls         `json:"tls,omitempty"`
	*DataStore   `json:"data_store,omitempty"`
	*JobStore    `json:"job_store,omitempty"`
	*DeviceStore `json:"device_store,omitempty"`
}

func NewDefault() *Config {
	return &Config{}
}

func NewFromFile(filename string) (*Config, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return NewFromBytes(bytes)
}

func NewFromBytes(bytes []byte) (*Config, error) {
	conf := new(Config)
	if err := json.Unmarshal(bytes, conf); err != nil {
		return nil, err
	}
	// TODO: extra validation
	return conf, nil
}

func (c *Config) ToBytes() ([]byte, error) {
	return json.Marshal(c)
}

func (c *Config) ToBytesPretty() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

type Tls struct {
	Enabled bool `json:"enabled,omitempty"`
}

type DataStore struct {
	Type          string `json:"type,omitempty"`
	*DataStoreGit `json:"git,omitempty"`
}

type DataStoreGit struct {
	Url                    string `json:"url,omitempty"`
	*DataStoreGitUser      `json:"user,omitempty"`
	PoolSize               int      `json:"pool_size,omitempty"`
	Structure              []string `json:"structure,omitempty"`
	IncludeReadmeOverviews bool     `json:"include_readme_overviews,omitempty"`
	DataDir                string   `json:"data_dir,omitempty"`
}

type DataStoreGitUser struct {
	FriendlyName string `json:"friendly_name,omitempty"`
	Email        string `json:"email,omitempty"`
	Name         string `json:"name,omitempty"`
	Pass         string `json:"pass,omitempty"`
}

type JobStore struct {
	Type           string `json:"type,omitempty"`
	*JobStoreLocal `json:"local,omitempty"`
}

type JobStoreLocal struct {
	JobGenerics map[string]*Job `json:"job_generics,omitempty"`
	Jobs        map[string]*Job `json:"jobs,omitempty"`
}

type Job struct {
	Generic        string `json:"generic,omitempty"`
	Type           string `json:"type,omitempty"`
	*JobSchedule   `json:"schedule,omitempty"`
	Commands       []*JobCommand       `json:"commands,omitempty"`
	CommandGeneric *JobCommand         `json:"command_generic,omitempty"`
	JobFile        map[string]*JobFile `json:"file,omitempty"`
	Scrubbers      []*JobScrubber      `json:"scrubbers,omitempty"`
	TemplateValues map[string]string   `json:"template_values,omitempty"`
}

type JobSchedule struct {
	Cron     string `json:"cron,omitempty"`
	Duration string `json:"duration,omitempty"`
	Iso8601  string `json:"iso_8601,omitempty"`
	Fixed    int64  `json:"fixed,omitempty"`
}

type JobCommand struct {
	Command       string   `json:"command,omitempty"`
	Expect        []string `json:"expect,omitempty"`
	ExpectNot     []string `json:"expect_not,omitempty"`
	Timeout       *int     `json:"timeout,omitempty"`
	ImplicitEnter *bool    `json:"implicit_enter,omitempty"`
}

type JobFile struct {
	Compression string `json:"compression,omitempty"`
}

type JobScrubber struct {
	Type    string `json:"type,omitempty"`
	Search  string `json:"search,omitempty"`
	Replace string `json:"replace,omitempty"`
}

type DeviceStore struct {
	Type              string `json:"type,omitempty"`
	*DeviceStoreLocal `json:"local,omitempty"`
}

type DeviceStoreLocal struct {
	DeviceGenerics map[string]*Device `json:"device_generics,omitempty"`
	Devices        map[string]*Device `json:"devices,omitempty"`
}

type Device struct {
	Generic            string `json:"generic,omitempty"`
	Host               string `json:"host,omitempty"`
	*DeviceProtocol    `json:"protocol,omitempty"`
	Tags               []string `json:"tags,omitempty"`
	*DeviceCredentials `json:"credentials,omitempty"`
	Jobs               map[string]*Job `json:"jobs,omitempty"`
}

type DeviceProtocol struct {
	Type               string `json:"type,omitempty"`
	*DeviceProtocolSsh `json:"ssh,omitempty"`
}

type DeviceProtocolSsh struct {
	Port              int  `json:"port,omitempty"`
	IncludeCbcCiphers bool `json:"include_cbc_ciphers,omitempty"`
}

type DeviceCredentials struct {
	User string `json:"user,omitempty"`
	Pass string `json:"pass,omitempty"`
}
