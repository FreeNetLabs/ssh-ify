package tunnel

import (
	"log"
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
		log.Printf("[session %s] No Upgrade header found. Closing connection.", s.sessionID)
		s.Close()
		return false
	}

	log.Printf("[session %s] WebSocket upgrade: using in-process SSH server.", s.sessionID)
	proxyEnd, sshEnd := net.Pipe()
	if s.sshConfig == nil {
		log.Printf("[session %s] SSH config not initialized", s.sessionID)
		s.Close()
		return false
	}
	go ssh.HandleSSHConnection(sshEnd, s.sshConfig, func() {
		s.server.Add(s)
	})
	s.target = proxyEnd
	if _, err := s.client.Write([]byte(WebSocketUpgradeResponse)); err != nil {
		log.Printf("[session %s] Failed to write WebSocket upgrade response: %v", s.sessionID, err)
		s.Close()
		return false
	}
	log.Printf("[session %s] Tunnel established.", s.sessionID)
	return true
}

func HeaderValue(headers []string, headerName string) string {
	headerNameLower := strings.ToLower(headerName)
	for _, line := range headers {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.ToLower(strings.TrimSpace(parts[0])) == headerNameLower {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}
