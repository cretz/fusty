package integration

import (
	"fmt"
	"gitlab.com/cretz/fusty/config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"
	"strconv"
	"io"
)

type mockDevice struct {
	username     string
	password     string
	listenOnHost string
	listenOnPort int
	serverConfig *ssh.ServerConfig
	listener     net.Listener
	responses    map[string]string
	prompt string
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
					log.Print("Login succeeded")
					return nil, nil
				}
				log.Print("Login failed")
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
	if m.listenOnHost == "" {
		m.listenOnHost = "127.0.0.1"
	}
	if m.listenOnPort == 0 {
		m.listenOnPort = 2223
	}
	if m.prompt == "" {
		m.prompt = "prompt> "
	}
	list, err := net.Listen("tcp", m.listenOnHost+":"+strconv.Itoa(m.listenOnPort))
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
		serverConn, chans, reqs, err := ssh.NewServerConn(conn, m.serverConfig)
		if err != nil {
			log.Printf("Failed to initiate client connection: %v", err)
			continue
		}
		go ssh.DiscardRequests(reqs)
		go m.handleChannels(chans, serverConn)
	}
}

func (m *mockDevice) handleChannels(chans <-chan ssh.NewChannel, serverConn *ssh.ServerConn) {
	for newChannel := range chans {
		go m.handleChannel(newChannel, serverConn)
	}
}

func (m *mockDevice) handleChannel(newCh ssh.NewChannel, serverConn *ssh.ServerConn) {
	if t := newCh.ChannelType(); t != "session" {
		log.Printf("Unknown channel type: %v", t)
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
					log.Print("Payload provided with shell")
					// We don't accept any commands, only the default shell
					ok = false
				}
			default:
				log.Printf("Unrecognized req type: %v", req.Type)
			}
			if req.WantReply {
				req.Reply(ok, nil)
			}
		}
	}(reqs)

	term := terminal.NewTerminal(ch, m.prompt)

	go func() {
		defer ch.Close()
		for {
			line, err := term.ReadLine()
			if err != nil && err != io.EOF {
				log.Printf("Unable to read line: %v", err)
				break
			}
			log.Printf("User typed: %v", line)
			if resp, ok := m.responses[line]; !ok {
				log.Print("Unrecognized line, ignoring")
			} else {
				log.Printf("Responding with: %v", resp)
				if _, err := term.Write([]byte(resp)); err != nil {
					log.Printf("Error responding: %v", err)
				}
			}
			if err == io.EOF {
				log.Print("EOF")
				break
			}
		}
	}()
}

func (m *mockDevice) stop() {
	if m.listener != nil {
		m.listener.Close()
	}
}

func (m *mockDevice) genericDevice() *config.Device {
	return &config.Device{
		Host: m.listenOnHost,
		DeviceProtocol: &config.DeviceProtocol{
			Type:              "ssh",
			DeviceProtocolSsh: &config.DeviceProtocolSsh{Port: m.listenOnPort},
		},
		DeviceCredentials: &config.DeviceCredentials{
			User: m.username,
			Pass: m.password,
		},
	}
}
