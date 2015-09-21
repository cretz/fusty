package integration

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"net"
)

type mockDevice struct {
	username     string
	password     string
	listenOn     string
	serverConfig *ssh.ServerConfig
	listener     net.Listener
	responses    map[string]string
}

// Some help from https://gist.github.com/jpillora/b480fde82bff51a06238 among other places

func (m *mockDevice) listen() error {
	if m.serverConfig == nil {
		m.serverConfig = &ssh.ServerConfig{}
		if m.username != nil {
			m.serverConfig.PasswordCallback = func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
				if c.User() == m.username && string(pass) == m.password {
					return nil, nil
				}
				return nil, fmt.Errorf("Password rejected for %q", c.User())
			}
		} else {
			m.serverConfig.NoClientAuth = true
		}
	}
	if m.listenOn == "" {
		m.listenOn = "127.0.0.1"
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
			log.Printf("Failed to connect client: %v", err)
			return nil
		}
		_, chans, reqs, err := ssh.NewServerConn(conn, m.serverConfig)
		if err != nil {
			log.Printf("Failed to connect client: %v", err)
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
			ok := false
			switch req.Type {
			case "shell":
				ok = true
				if len(req.Payload) > 0 {
					ok = false
				}
			}
			req.Reply(ok, nil)
		}
	}(reqs)
	term := terminal.NewTerminal(ch, "> ")
	go func() {
		defer ch.Close()
		for {
			line, err := term.ReadLine()
			if err != nil {
				break
			}
			fmt.Printf("YOU WROTE: %v\n", line)
		}
	}()
}

func (m *mockDevice) stop() {
	if m.listener != nil {
		m.listener.Close()
	}
}
