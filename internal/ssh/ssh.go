package ssh

import (
	"fmt"
	"net"
	"os"

	"github.com/FreeNetLabs/ssh-ify/internal/config"
	"golang.org/x/crypto/ssh"
)

type ServerConfig = ssh.ServerConfig

func NewConfig(cfg *config.Config) (*ServerConfig, error) {
	users := make(map[string]string)
	for _, u := range cfg.Users {
		if u.Name != "" && u.Password != "" {
			users[u.Name] = u.Password
		}
	}

	privateBytes, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		if err := GenerateHostKey(cfg.KeyPath); err != nil {
			return nil, fmt.Errorf("failed to generate host key: %v", err)
		}
		privateBytes, err = os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read generated host key: %v", err)
		}
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %v", err)
	}

	cfgSSH := &ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if expected, ok := users[c.User()]; ok && expected == string(pass) {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
		BannerCallback: func(conn ssh.ConnMetadata) string {
			return cfg.Banner
		},
		ServerVersion: "SSH-2.0-ssh-ify",
	}

	cfgSSH.AddHostKey(private)
	return cfgSSH, nil
}

func HandleConnection(conn net.Conn, config *ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		conn.Close()
		return
	}
	go ssh.DiscardRequests(reqs)
	HandleChannels(chans)
	sshConn.Close()
}
