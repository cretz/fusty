package integration

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/cretz/fusty/config"
	"golang.org/x/crypto/ssh"
	"strconv"
)

type juniperDevice struct {
	host                  string
	sshPort               int
	sshUser               string
	sshPassword           string
	configFile            string
	configFileCompression string
}

func newDefaultEmulatedJuniperDevice() *juniperDevice {
	return &juniperDevice{
		host:                  "127.0.0.1",
		sshPort:               2222,
		sshUser:               "root",
		sshPassword:           "Juniper",
		configFile:            "/config/juniper.conf.gz",
		configFileCompression: "gzip",
	}
}

func (j *juniperDevice) assertOnline() {
	sshConf := &ssh.ClientConfig{
		User: j.sshUser,
		Auth: []ssh.AuthMethod{ssh.Password(j.sshPassword)},
	}
	client, err := ssh.Dial("tcp", j.host+":"+strconv.Itoa(j.sshPort), sshConf)
	So(err, ShouldBeNil)
	defer client.Close()
	session, err := client.NewSession()
	So(err, ShouldBeNil)
	session.Close()
}

func (j *juniperDevice) genericJob() *config.Job {
	return &config.Job{
		Type: "file",
		JobFile: map[string]*config.JobFile{
			j.configFile: &config.JobFile{Compression: j.configFileCompression},
		},
	}
}

func (j *juniperDevice) genericDevice() *config.Device {
	return &config.Device{
		Host: j.host,
		DeviceProtocol: &config.DeviceProtocol{
			Type:              "ssh",
			DeviceProtocolSsh: &config.DeviceProtocolSsh{Port: j.sshPort},
		},
		DeviceCredentials: &config.DeviceCredentials{
			User: j.sshUser,
			Pass: j.sshPassword,
		},
	}
}
