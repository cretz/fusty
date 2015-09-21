package model

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/config"
)

type Device struct {
	Name               string `json:"name"`
	Host               string `json:"host"`
	*DeviceCredentials `json:"credentials"`
	*DeviceProtocol    `json:"protocol"`
	Tags               []string        `json:"-"`
	Jobs               map[string]*Job `json:"-"`
}

func NewDefaultDevice(name string) *Device {
	return &Device{
		Name:           name,
		Host:           name,
		DeviceProtocol: &DeviceProtocol{Type: "ssh", SshDeviceProtocol: &SshDeviceProtocol{Port: 22}},
	}
}

func (d *Device) ApplyConfig(conf *config.Device) error {
	if conf.Host != "" {
		d.Host = conf.Host
	}
	if conf.DeviceProtocol != nil {
		switch conf.DeviceProtocol.Type {
		case "ssh":
			d.DeviceProtocol.Type = "ssh"
			port := 22
			if conf.DeviceProtocol.DeviceProtocolSsh != nil {
				port = conf.DeviceProtocol.DeviceProtocolSsh.Port
			}
			d.DeviceProtocol.SshDeviceProtocol = &SshDeviceProtocol{Port: port}
		default:
			return fmt.Errorf("Unrecognized protocol type: %v", conf.Type)
		}
	}
	d.Tags = append(d.Tags, conf.Tags...)
	if conf.DeviceCredentials != nil {
		if d.DeviceCredentials == nil {
			d.DeviceCredentials = &DeviceCredentials{}
		}
		if conf.DeviceCredentials.User != "" {
			d.DeviceCredentials.User = conf.DeviceCredentials.User
		}
		if conf.DeviceCredentials.Pass != "" {
			d.DeviceCredentials.Pass = conf.DeviceCredentials.Pass
		}
	}
	// We expect the job to be present to overwrite it with anything
	for name, job := range conf.Jobs {
		if existing, ok := d.Jobs[name]; ok {
			if err := existing.ApplyConfig(job); err != nil {
				return fmt.Errorf("Unable to configure job %v: %v", name, err)
			}
		}
	}
	return nil
}

func (d *Device) Validate() []error {
	errs := []error{}
	if d.Host == "" {
		errs = append(errs, errors.New("Host required"))
	}
	if d.DeviceProtocol == nil {
		errs = append(errs, errors.New("Protocol required"))
	}
	// TODO: validate credentials
	for name, job := range d.Jobs {
		for _, err := range job.Validate() {
			errs = append(errs, fmt.Errorf("Invalid job %v: %v", name, err))
		}
	}
	return errs
}

type DeviceProtocol struct {
	Type               string `json:"type"`
	*SshDeviceProtocol `json:"ssh,omitempty"`
}

type SshDeviceProtocol struct {
	Port int `json:"port"`
}

type DeviceCredentials struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}
