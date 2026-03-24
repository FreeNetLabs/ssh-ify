package proxy

import (
	"log"
	"net"

	"github.com/FreeNetLabs/ssh-ify/internal/ssh"
)

type Conn struct {
	client    net.Conn
	sshServer *ssh.Server
}

func (c *Conn) Serve() {
	buf := make([]byte, 4096)
	n, err := c.client.Read(buf)
	if err != nil {
		c.Close()
		return
	}

	reqData := string(buf[:n])

	if !IsWebSocketUpgrade(reqData) {
		c.client.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK"))
		c.Close()
		return
	}

	if err := UpgradeWebSocket(c); err != nil {
		log.Printf("websocket upgrade err: %v", err)
		c.Close()
		return
	}

	c.sshServer.HandleConnection(c.client)
}

func (c *Conn) Close() {
	if c.client != nil {
		c.client.Close()
	}
}
