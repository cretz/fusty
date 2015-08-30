package model

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/config"
)

type Device struct {
	Name string `json:"name"`
	Host string `json:"ip"`
	DeviceProtocol
	Tags []string `json:"tags"`
	*DeviceCredentials
	Jobs map[string]*Job `json:"jobs"`
}

func NewDefaultDevice(name string) *Device {
	return &Device{
		Name:           name,
		Host:           name,
		DeviceProtocol: &SshDeviceProtocol{Port: 22},
	}
}

func (d *Device) ApplyConfig(conf *config.Device) error {
	if conf.Host != "" {
		d.Host = conf.Host
	}
	if conf.DeviceProtocol != nil {
		switch conf.DeviceProtocol.Type {
		case "ssh":
			if _, ok := d.DeviceProtocol.(SshDeviceProtocol); !ok {
				d.DeviceProtocol = &SshDeviceProtocol{Port: 22}
			}
			if conf.DeviceProtocolSsh.Port > 0 {
				d.DeviceProtocol.(*SshDeviceProtocol).Port = conf.Port
			}
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
		if conf.DeviceCredentials.Prompt != nil {
			if d.Prompt == nil {
				d.Prompt = NewDefaultPrompt()
			}
			if err := d.Prompt.ApplyConfig(conf.DeviceCredentials.Prompt); err != nil {
				return fmt.Errorf("Error building prompt: %v", err)
			}
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

type DeviceProtocol interface {
}

type SshDeviceProtocol struct {
	Port int
}

type DeviceCredentials struct {
	User string
	Pass string
	*Prompt
}
