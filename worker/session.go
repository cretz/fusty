package worker

import (
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"gitlab.com/cretz/fusty/model"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"time"
)

type session interface {
	authenticate(device *model.Device) error

	// Should be called even on auth failure
	close() error

	// Note, both bytes and error can be set
	run(cmd string) ([]byte, error)

	fetchFile(path string) ([]byte, error)

	shell() (sessionShell, error)
}

type sessionShell interface {
	io.Reader
	io.Writer
	close() error
	// Error can be eof
	readFor(duration time.Duration) ([]byte, error)
}

func newSession(device *model.Device) (session, error) {
	if device.DeviceProtocol.SshDeviceProtocol == nil {
		return nil, errors.New("Unable to find SSH settings")
	}
	return &sshSession{}, nil
}

type sshSession struct {
	device *model.Device
	client *ssh.Client
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
	s.device = device
	s.client = client
	return nil
}

func (s *sshSession) close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

func (s *sshSession) run(cmd string) ([]byte, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Unable to initiate session on %v: %v", s.device.Host, err)
	}
	defer session.Close()
	if Verbose {
		log.Printf("Running SSH command on %v: %v", s.device.Host, cmd)
	}
	return session.CombinedOutput(cmd)
}

func (s *sshSession) fetchFile(path string) ([]byte, error) {
	client, err := sftp.NewClient(s.client)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to SFTP on %v: %v", s.device.Host, err)
	}
	defer client.Close()
	file, err := client.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Unable to open %v via SFTP on %v: %v", path, s.device.Host, err)
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("Unable to read %v via SFTP on %v: %v", path, s.device.Host, err)
	}
	return bytes, nil
}

func (s *sshSession) shell() (sessionShell, error) {
	sess, err := s.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Unable to initiate session on %v: %v", s.device.Host, err)
	}
	sshOut, err := sess.StdoutPipe()
	if err != nil {
		sess.Close()
		return nil, fmt.Errorf("Unable to open stdout pipe to external server: %v", err)
	}
	sshErr, err := sess.StderrPipe()
	if err != nil {
		sess.Close()
		return nil, fmt.Errorf("Unable to open stderr pipe to external server: %v", err)
	}
	sshIn, err := sess.StdinPipe()
	if err != nil {
		sess.Close()
		return nil, fmt.Errorf("Unable to open stdin pipe to external server: %v", err)
	}
	// TODO: what about request pty goodies?
	return &sshSessionShell{
		Reader:          io.MultiReader(sshOut, sshErr),
		WriteCloser:     sshIn,
		internalSession: sess,
	}, nil
}

type sshSessionShell struct {
	io.Reader
	io.WriteCloser
	internalSession *ssh.Session
}

func (s *sshSessionShell) close() error {
	ret := s.WriteCloser.Close()
	if err := s.internalSession.Close(); err != nil {
		ret = err
	}
	return fmt.Errorf("Unable to close shell: %v", ret)
}

func (s *sshSessionShell) readFor(duration time.Duration) ([]byte, error) {
	byteChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)
	quitChan := make(chan bool, 1)
	go func() {
		tempBuf := make([]byte, 500)
		for {
			select {
			case <-quitChan:
				return
			default:
				n, err := s.Read(tempBuf)
				if n > 0 {
					currSet := make([]byte, n)
					copy(currSet, tempBuf)
					byteChan <- currSet
				}
				if err != nil {
					errorChan <- err
					return
				}
			}
		}
	}()
	retBytes := []byte{}
	timeoutChan := time.After(duration)
	for {
		select {
		case buf := <-byteChan:
			retBytes = append(retBytes, buf...)
		case <-timeoutChan:
			quitChan <- true
			return retBytes, nil
		case err := <-errorChan:
			return retBytes, err
		}
	}
	panic("Can't get here")
}