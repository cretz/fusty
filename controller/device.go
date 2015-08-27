package controller

type Device struct {
	Name string
	Host string
	DeviceProtocol
	Tags []string
	*DeviceCredentials
	Jobs []*Job
}

type DeviceProtocol interface {
}

type DeviceCredentials struct {
	User   string
	Pass   string
	Prompt string
}
