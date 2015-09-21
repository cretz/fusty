package integration

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"
	"strings"
)

type mockDevice struct {
	username         string
	password         string
	listenOn         string
	serverConfig     *ssh.ServerConfig
	listener         net.Listener
	responses        map[string]string
	responseStatuses map[string]int
}

// Some help from https://gist.github.com/jpillora/b480fde82bff51a06238 and
// https://godoc.org/golang.org/x/crypto/ssh#example-NewServerConn and other places

func (m *mockDevice) listen() error {
	if m.serverConfig == nil {
		m.serverConfig = &ssh.ServerConfig{}
		if m.username != "" {
			m.serverConfig.PasswordCallback = func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
				log.Printf("SSH auth being attempted by %v", c.User())
				if c.User() == m.username && string(pass) == m.password {
					return nil, nil
				}
				return nil, fmt.Errorf("Password rejected for %q", c.User())
			}
		} else {
			m.serverConfig.NoClientAuth = true
		}
		privateBytes, err := ioutil.ReadFile(filepath.Join(baseDirectory, "integration", "mock_device_id_rsa"))
		if err != nil {
			return fmt.Errorf("Unable to fetch private key: %v", err)
		}
		private, err := ssh.ParsePrivateKey(privateBytes)
		if err != nil {
			return fmt.Errorf("Unable to parse private key: %v", err)
		}
		m.serverConfig.AddHostKey(private)
	}
	if m.listenOn == "" {
		m.listenOn = "127.0.0.1:0"
	}
	list, err := net.Listen("tcp", m.listenOn)
	if err != nil {
		return fmt.Errorf("Unable to start mock device: %v", err)
	}
	m.listener = list
	return nil
}

func (m *mockDevice) addr() *net.TCPAddr {
	return m.listener.Addr().(*net.TCPAddr)
}

func (m *mockDevice) acceptUntilError() error {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			log.Printf("Failed to accept client: %v", err)
			return nil
		}
		_, chans, reqs, err := ssh.NewServerConn(conn, m.serverConfig)
		if err != nil {
			log.Printf("Failed to initiate client connection: %v", err)
			continue
		}
		go ssh.DiscardRequests(reqs)
		go m.handleChannels(chans)
	}
}

func (m *mockDevice) handleChannels(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		go m.handleChannel(newChannel)
	}
}

func (m *mockDevice) handleChannel(newCh ssh.NewChannel) {
	if t := newCh.ChannelType(); t != "session" {
		newCh.Reject(ssh.UnknownChannelType, fmt.Sprintf("Unknown channel type: %v", t))
		return
	}
	ch, reqs, err := newCh.Accept()
	if err != nil {
		log.Printf("Could not accept channel: %v", err)
		return
	}
	go func(in <-chan *ssh.Request) {
		for req := range in {
			log.Printf("GOT REQ TYPE: %v, WANT REPLY %v, BODY: %v", req.Type, req.WantReply, string(req.Payload))
			ok := false
			switch req.Type {
			// TODO: support shell
			case "exec":
				defer ch.Close()
				ok = true
				payload := bytes.Trim(req.Payload, "\x00\x08")
				cmd := strings.TrimSpace(string(payload))
				log.Printf("Handling SSH command: '%v'", cmd)
				// TODO: actually handle it properly here, I don't know how but may not need to since
				// shell is the more important one anyways
			}
			req.Reply(ok, nil)
		}
	}(reqs)
}

func (m *mockDevice) stop() {
	if m.listener != nil {
		m.listener.Close()
	}
}
