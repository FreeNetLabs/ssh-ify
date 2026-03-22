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

func UpgradeWebSocket(s *Session, reqLines []string) bool {
	upgradeHeader := getHeaderValue(reqLines, "Upgrade")

	if upgradeHeader == "" {
		s.Close()
		return false
	}

	proxyEnd, sshEnd := net.Pipe()

	go ssh.HandleConnection(sshEnd, s.sshConfig)
	s.target = proxyEnd

	if _, err := s.client.Write([]byte(WebSocketUpgradeResponse)); err != nil {
		s.Close()
		return false
	}
	return true
}

func getHeaderValue(headers []string, headerName string) string {
	headerNameLower := strings.ToLower(headerName)
	for _, line := range headers {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
		if len(parts) == 2 && strings.ToLower(strings.TrimSpace(parts[0])) == headerNameLower {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}
