package worker

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/model"
	"golang.org/x/crypto/ssh"
	"log"
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
	if device.DeviceProtocol.SshDeviceProtocol == nil {
		return nil, errors.New("Unable to find SSH settings")
	}
	return &sshSession{}, nil
}

type sshSession struct {
	device  *model.Device
	session *ssh.Session
}

func (s *sshSession) authenticate(device *model.Device) error {
	sshConf := &ssh.ClientConfig{
		User: device.DeviceCredentials.User,
		Auth: []ssh.AuthMethod{ssh.Password(device.DeviceCredentials.Pass)},
	}
	hostPort := device.Host + ":" + strconv.Itoa(device.DeviceProtocol.SshDeviceProtocol.Port)
	if Verbose {
		log.Printf("Starting SSH session on %v for user %v", hostPort, device.DeviceCredentials.User)
	}
	client, err := ssh.Dial("tcp", hostPort, sshConf)
	if err != nil {
		return fmt.Errorf("Unable to connect to %v: %v", hostPort, err)
	}
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("Unable to initiate session on %v: %v", hostPort, err)
	}
	s.device = device
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
	if Verbose {
		log.Printf("Running SSH command on %v: %v", s.device.Host, cmd)
	}
	return s.session.CombinedOutput(cmd)
}
