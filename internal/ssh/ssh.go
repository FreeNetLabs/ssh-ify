package ssh

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/FreeNetLabs/ssh-ify/internal/config"
	"golang.org/x/crypto/ssh"
)

type ServerConfig = ssh.ServerConfig

var userCredentials map[string]string

func NewConfig(cfg *config.Config) (*ssh.ServerConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration required for SSH server config")
	}

	if err := InitializeAuth(cfg); err != nil {
		return nil, err
	}

	keyPath := cfg.SSHHostKeyPath

	privateBytes, err := os.ReadFile(keyPath)
	if err != nil {
		privateKey, err := NewRSAPrivateKey(4096)
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %v", err)
		}

		privateBytes = RSAPrivateKeyPEM(privateKey)
		if err := os.WriteFile(keyPath, privateBytes, 0600); err != nil {
			return nil, fmt.Errorf("failed to save generated host key: %v", err)
		}
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse host key: %v", err)
	}

	cfgSSH := &ssh.ServerConfig{
		PasswordCallback: PasswordAuth,
		BannerCallback: func(conn ssh.ConnMetadata) string {
			if cfg.Banner != "" {
				return cfg.Banner
			}
			return config.DefaultBanner
		},
	}

	cfgSSH.ServerVersion = "SSH-2.0-ssh-ify_1.0"
	cfgSSH.AddHostKey(private)
	return cfgSSH, nil
}

func InitializeAuth(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration required for auth initialization")
	}

	userCredentials = make(map[string]string)
	for _, u := range cfg.Users {
		if u.Username == "" || u.Password == "" {
			continue
		}
		userCredentials[u.Username] = u.Password
	}

	if len(userCredentials) == 0 {
		return fmt.Errorf("no users configured: set users in config file")
	}

	return nil
}

func PasswordAuth(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	if userCredentials == nil {
		log.Printf("PasswordAuth: auth is not initialized")
		return nil, fmt.Errorf("authentication not initialized")
	}

	expected, exists := userCredentials[c.User()]
	if !exists || expected != string(password) {
		log.Printf("PasswordAuth: failed login attempt for user '%s'", c.User())
		return nil, fmt.Errorf("invalid credentials")
	}

	log.Printf("PasswordAuth: successful login for user '%s'", c.User())
	return nil, nil
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
