package proxy

import (
	"fmt"
	"strings"
)

const (
	WebSocketUpgradeResponse = "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n\r\n"
)

func UpgradeWebSocket(c *Conn) error {
	if _, err := c.client.Write([]byte(WebSocketUpgradeResponse)); err != nil {
		c.Close()
		return fmt.Errorf("write upgrade resp: %w", err)
	}
	return nil
}

func IsWebSocketUpgrade(req string) bool {
	return strings.Contains(strings.ToLower(req), "upgrade: websocket")
}
