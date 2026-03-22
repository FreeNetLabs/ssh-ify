package tunnel

import (
	"net"
	"strings"

	"github.com/FreeNetLabs/ssh-ify/internal/ssh"
)

const (
	WebSocketUpgradeResponse = "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n\r\n"
)

func WebSocketHandler(s *Session, reqLines []string) bool {
	upgradeHeader := HeaderValue(reqLines, "Upgrade")

	if upgradeHeader == "" {
		s.Close()
		return false
	}

	proxyEnd, sshEnd := net.Pipe()

	go ssh.HandleSSHConnection(sshEnd, s.sshConfig, func() {})
	s.target = proxyEnd

	if _, err := s.client.Write([]byte(WebSocketUpgradeResponse)); err != nil {
		s.Close()
		return false
	}
	return true
}

func HeaderValue(headers []string, headerName string) string {
	headerNameLower := strings.ToLower(headerName)
	for _, line := range headers {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
		if len(parts) == 2 && strings.ToLower(strings.TrimSpace(parts[0])) == headerNameLower {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}
