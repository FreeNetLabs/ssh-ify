package ssh

import (
	"fmt"
	"net"

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

	private, err := loadHostKey()
	if err != nil {
		return nil, fmt.Errorf("load host key: %v", err)
	}

	sshCfg := &ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if expected, ok := users[c.User()]; ok && expected == string(pass) {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid creds")
		},
		BannerCallback: func(conn ssh.ConnMetadata) string {
			return cfg.Banner
		},
		ServerVersion: "SSH-2.0-ssh-ify",
	}

	sshCfg.AddHostKey(private)
	return sshCfg, nil
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
