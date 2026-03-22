package proxy

import (
	"fmt"
	"net"
	"strings"

	"github.com/FreeNetLabs/ssh-ify/internal/ssh"
)

const (
	WebSocketUpgradeResponse = "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n\r\n"
)

func UpgradeWebSocket(c *Conn) error {
	proxyEnd, sshEnd := net.Pipe()

	go ssh.HandleConnection(sshEnd, c.sshCfg)
	c.target = proxyEnd

	if _, err := c.client.Write([]byte(WebSocketUpgradeResponse)); err != nil {
		c.Close()
		return fmt.Errorf("failed to write upgrade response: %w", err)
	}
	return nil
}

func IsWebSocketUpgrade(req string) bool {
	return strings.Contains(strings.ToLower(req), "upgrade: websocket")
}
