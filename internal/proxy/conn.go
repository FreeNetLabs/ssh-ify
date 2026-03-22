package proxy

import (
	"io"
	"net"

	"github.com/FreeNetLabs/ssh-ify/internal/ssh"
)

type Conn struct {
	client net.Conn
	target net.Conn
	sshCfg *ssh.ServerConfig
}

func (c *Conn) Close() {
	if c.client != nil {
		c.client.Close()
	}
	if c.target != nil {
		c.target.Close()
	}
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

	if UpgradeWebSocket(c) == nil {
		c.Proxy()
	} else {
		c.Close()
	}
}

func (c *Conn) Proxy() {
	defer c.Close()

	go func() {
		io.Copy(c.target, c.client)
		c.target.Close()
	}()
	io.Copy(c.client, c.target)
}
