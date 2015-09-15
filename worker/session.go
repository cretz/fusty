package worker

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/model"
	"golang.org/x/crypto/ssh"
	"strconv"
)

type session interface {
	authenticate(device *model.Device) error

	// Should be called even on auth failure
	close() error

	// Note, both bytes and error can be set
	run(cmd string) ([]byte, error)
}

func newSession(device *model.Device) (session, error) {
	switch device.DeviceProtocol.(type) {
	case model.SshDeviceProtocol:
		return &sshSession{}, nil
	default:
		return nil, errors.New("Unrecognized device protocol")
	}
}

type sshSession struct {
	session *ssh.Session
}

func (s *sshSession) authenticate(device *model.Device) error {
	sshProtocol, ok := device.DeviceProtocol.(model.SshDeviceProtocol)
	if !ok {
		return errors.New("Invalid protocol")
	}
	sshConf := &ssh.ClientConfig{
		User: device.DeviceCredentials.User,
		Auth: []ssh.AuthMethod{ssh.Password(device.DeviceCredentials.Pass)},
	}
	hostPort := device.Host + ":" + strconv.Itoa(sshProtocol.Port)
	client, err := ssh.Dial("tcp", hostPort, sshConf)
	if err != nil {
		return fmt.Errorf("Unable to connect to %v: %v", hostPort, err)
	}
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("Unable to initiate session on %v: %v", hostPort, err)
	}
	s.session = session
	return nil
}

func (s *sshSession) close() error {
	if s.session == nil {
		return nil
	}
	return s.session.Close()
}

func (s *sshSession) run(cmd string) ([]byte, error) {
	if s.session == nil {
		return nil, errors.New("Session not started")
	}
	return s.session.CombinedOutput(cmd)
}
