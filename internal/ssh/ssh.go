package ssh

import (
	"fmt"
	"net"
	"os"

	"github.com/FreeNetLabs/ssh-ify/internal/config"
	"golang.org/x/crypto/ssh"
)

type ServerConfig = ssh.ServerConfig

func NewConfig(cfg *config.Config) (*ssh.ServerConfig, error) {
	users := make(map[string]string)
	for _, u := range cfg.Users {
		if u.Username != "" && u.Password != "" {
			users[u.Username] = u.Password
		}
	}

	privateBytes, err := os.ReadFile(cfg.SSHHostKeyPath)
	if err != nil {
		if err := GenerateHostKey(cfg.SSHHostKeyPath); err != nil {
			return nil, fmt.Errorf("failed to generate host key: %v", err)
		}
		privateBytes, err = os.ReadFile(cfg.SSHHostKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read generated host key: %v", err)
		}
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %v", err)
	}

	cfgSSH := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if expected, ok := users[c.User()]; ok && expected == string(pass) {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
		BannerCallback: func(conn ssh.ConnMetadata) string {
			return cfg.Banner
		},
		ServerVersion: "SSH-2.0-ssh-ify_1.0",
	}

	cfgSSH.AddHostKey(private)
	return cfgSSH, nil
}

func HandleSSHConnection(conn net.Conn, config *ssh.ServerConfig, onAuthSuccess func()) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		conn.Close()
		return
	}
	if onAuthSuccess != nil {
		onAuthSuccess()
	}
	go ssh.DiscardRequests(reqs)
	HandleSSHChannels(chans)
	sshConn.Close()
}
