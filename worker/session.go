package worker

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/model"
	// Does not include CBC ciphers, ref: https://groups.google.com/forum/#!topic/golang-nuts/J2XCsTsNQ9o
	// TODO: decide if we like this guy's fork or if I should make my own
	// "golang.org/x/crypto/ssh"
	// "github.com/pkg/sftp"
	"bytes"
	"github.com/ScriptRock/crypto/ssh"
	"github.com/ScriptRock/sftp"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"sync"
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
	io.Writer
	close() error
	bytesAndReset() []byte
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
	if device.DeviceProtocol.SshDeviceProtocol.IncludeCbcCiphers {
		sshConf.Config = ssh.Config{Ciphers: ssh.AllSupportedCiphers()}
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
	stdOutAndErrBuff := newThreadSafeByteBuffer()
	sess.Stdout = stdOutAndErrBuff
	sess.Stderr = stdOutAndErrBuff
	sshIn, err := sess.StdinPipe()
	if err != nil {
		sess.Close()
		return nil, fmt.Errorf("Unable to open stdin pipe on %v: %v", s.device.Host, err)
	}
	modes := ssh.TerminalModes{}
	if err := sess.RequestPty("dumb", 80, 40, modes); err != nil {
		sess.Close()
		return nil, fmt.Errorf("Unable to request pty on %v: %v", s.device.Host, err)
	}
	if err := sess.Shell(); err != nil {
		sess.Close()
		return nil, fmt.Errorf("Unable to start shell on %v: %v", s.device.Host, err)
	}
	// TODO: what about request pty goodies?
	return &sshSessionShell{
		WriteCloser:      sshIn,
		internalSession:  sess,
		stdOutAndErrBuff: stdOutAndErrBuff,
	}, nil
}

type sshSessionShell struct {
	io.WriteCloser
	internalSession  *ssh.Session
	stdOutAndErrBuff *threadSafeByteBuffer
}

func (s *sshSessionShell) close() error {
	ret := s.WriteCloser.Close()
	if err := s.internalSession.Close(); err != nil {
		ret = err
	}
	return fmt.Errorf("Unable to close shell: %v", ret)
}

func (s *sshSessionShell) bytesAndReset() []byte {
	return s.stdOutAndErrBuff.bytesAndReset()
}

type threadSafeByteBuffer struct {
	buff *bytes.Buffer
	lock *sync.Mutex
}

func newThreadSafeByteBuffer() *threadSafeByteBuffer {
	return &threadSafeByteBuffer{
		buff: &bytes.Buffer{},
		lock: &sync.Mutex{},
	}
}

func (t *threadSafeByteBuffer) Write(p []byte) (n int, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.buff.Write(p)
}

func (t *threadSafeByteBuffer) bytesAndReset() []byte {
	t.lock.Lock()
	defer t.lock.Unlock()
	// We want this to run first
	defer t.buff.Reset()
	return t.buff.Bytes()
}
