package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/ayanrajpoot10/ssh-ify/internal/config"
	"golang.org/x/crypto/ssh"
)

const (
	SSHBufferPoolSize = 32 * 1024
)

type ServerConfig = ssh.ServerConfig

var (
	userCredentials map[string]string

	sshBufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, SSHBufferPoolSize)
			return &buf
		},
	}
)

func getSSHBuffer() *[]byte {
	return sshBufferPool.Get().(*[]byte)
}

func putSSHBuffer(buf *[]byte) {
	sshBufferPool.Put(buf)
}

func CopyWithSSHBuffer(dst io.Writer, src io.Reader) (int64, error) {
	buf := getSSHBuffer()
	defer putSSHBuffer(buf)
	return io.CopyBuffer(dst, src, *buf)
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
		return fmt.Errorf("no users configured: set users in config file or SSH_IFY_USERS env")
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

func NewRSAPrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}
	if err := privateKey.Validate(); err != nil {
		return nil, err
	}
	return privateKey, nil
}

func RSAPrivateKeyPEM(privateKey *rsa.PrivateKey) []byte {
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	}
	return pem.EncodeToMemory(privBlock)
}

func NewConfig(cfg *config.Config) (*ssh.ServerConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration required for SSH server config")
	}

	if err := InitializeAuth(cfg); err != nil {
		return nil, err
	}

	keyPath := cfg.SSHHostKeyPath
	if keyPath == "" {
		keyPath = config.DefaultSSHHostKeyPath
	}

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
			return "Welcome to ssh-ify.\n"
		},
	}

	cfgSSH.ServerVersion = "SSH-2.0-ssh-ify_1.0"
	cfgSSH.AddHostKey(private)
	return cfgSSH, nil
}

func ForwardData(ch ssh.Channel, targetConn net.Conn, addr string) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := CopyWithSSHBuffer(targetConn, ch)
		if err != nil && err != io.EOF {
			log.Printf("forwardChannel: Error copying SSH->%s: %v", addr, err)
		}
	}()
	go func() {
		defer wg.Done()
		_, err := CopyWithSSHBuffer(ch, targetConn)
		if err != nil && err != io.EOF {
			log.Printf("forwardChannel: Error copying %s->SSH: %v", addr, err)
		}
	}()
	wg.Wait()
	targetConn.Close()
	ch.Close()
}

func HandleSSHChannels(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		if !isDirectTCPIPChannel(newChannel) {
			log.Printf("HandleChannels: Unknown channel type: %s", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "only port forwarding allowed")
			continue
		}

		targetHost, targetPort, err := parseDirectTCPIPExtra(newChannel.ExtraData())
		if err != nil {
			log.Printf("HandleChannels: %v", err)
			newChannel.Reject(ssh.Prohibited, err.Error())
			continue
		}

		ch, reqs, err := newChannel.Accept()
		if err != nil {
			log.Printf("HandleChannels: Error accepting channel: %v", err)
			continue
		}
		go ssh.DiscardRequests(reqs)
		go handlePortForwarding(targetHost, targetPort, ch)
	}
}

func isDirectTCPIPChannel(newChannel ssh.NewChannel) bool {
	return newChannel.ChannelType() == "direct-tcpip"
}

func parseDirectTCPIPExtra(extra []byte) (string, uint32, error) {
	if len(extra) < 4 {
		return "", 0, fmt.Errorf("invalid direct-tcpip request: insufficient data for host length")
	}
	l := int(binary.BigEndian.Uint32(extra[:4]))
	if len(extra) < 4+l+4 {
		return "", 0, fmt.Errorf("invalid direct-tcpip request: insufficient data for host and port")
	}
	targetHost := string(extra[4 : 4+l])
	portOffset := 4 + l
	targetPort := binary.BigEndian.Uint32(extra[portOffset : portOffset+4])
	return targetHost, targetPort, nil
}

func handlePortForwarding(targetHost string, targetPort uint32, ch ssh.Channel) {
	defer ch.Close()
	addr := net.JoinHostPort(targetHost, strconv.Itoa(int(targetPort)))
	targetConn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("HandleChannels: Error connecting to target %s: %v", addr, err)
		return
	}
	ForwardData(ch, targetConn, addr)
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
