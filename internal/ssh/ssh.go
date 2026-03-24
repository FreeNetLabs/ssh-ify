package ssh

import (
	"fmt"
	"net"

	"github.com/FreeNetLabs/ssh-ify/internal/config"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	sshCfg *ssh.ServerConfig
}

func NewServer(cfg *config.Config) (*Server, error) {
	sshCfg, err := NewConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Server{sshCfg: sshCfg}, nil
}

func NewConfig(cfg *config.Config) (*ssh.ServerConfig, error) {
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

	sshCfg := &ssh.ServerConfig{
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

func (s *Server) HandleConnection(conn net.Conn) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.sshCfg)
	if err != nil {
		conn.Close()
		return
	}
	go ssh.DiscardRequests(reqs)
	s.HandleChannels(chans)
	sshConn.Close()
}
