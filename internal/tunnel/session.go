package tunnel

import (
	"bufio"
	"io"
	"net"
	"strings"

	"github.com/FreeNetLabs/ssh-ify/internal/ssh"
)

type Session struct {
	client    net.Conn
	target    net.Conn
	sshConfig *ssh.ServerConfig
}

func (s *Session) Close() {
	if s.client != nil {
		s.client.Close()
	}
	if s.target != nil {
		s.target.Close()
	}
}

func (s *Session) Serve() {
	reader := bufio.NewReader(s.client)
	var builder strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			s.Close()
			return
		}
		builder.WriteString(line)
		if strings.HasSuffix(builder.String(), "\r\n\r\n") {
			break
		}
	}
	buf := builder.String()
	reqLines := strings.Split(buf, "\r\n")

	if UpgradeWebSocket(s, reqLines) {
		s.Proxy()
	}
}

func (s *Session) Proxy() {
	defer s.Close()

	go func() {
		io.Copy(s.target, s.client)
		s.target.Close()
	}()
	io.Copy(s.client, s.target)
}
