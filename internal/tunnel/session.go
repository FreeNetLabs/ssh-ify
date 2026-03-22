package tunnel

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/FreeNetLabs/ssh-ify/internal/ssh"
)

const (
	ClientReadTimeout = 60 * time.Second
	MaxHeaderSize     = 16384
)

type Session struct {
	client    net.Conn
	target    net.Conn
	server    *Server
	sshConfig *ssh.ServerConfig
	sessionID string
}

func (s *Session) Close() {
	if s.client != nil {
		s.client.Close()
	}
	if s.target != nil {
		s.target.Close()
	}
}

func (s *Session) Handle() {
	log.Printf("[session %s] New connection opened", s.sessionID)

	s.client.SetReadDeadline(time.Now().Add(ClientReadTimeout))
	reader := bufio.NewReader(s.client)
	var builder strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("[session %s] Error reading from client: %v", s.sessionID, err)
			log.Printf("[session %s] Closing connection due to read error.", s.sessionID)
			return
		}
		builder.WriteString(line)
		if strings.HasSuffix(builder.String(), "\r\n\r\n") {
			break
		}
		if builder.Len() > MaxHeaderSize {
			log.Printf("[session %s] Header too large, closing connection", s.sessionID)
			s.client.Write([]byte("HTTP/1.1 431 Request Header Fields Too Large\r\n\r\n"))
			return
		}
	}
	buf := builder.String()

	reqLines := strings.Split(buf, "\r\n")
	if len(reqLines) > 0 {
		log.Printf("[session %s] Request received: %s", s.sessionID, reqLines[0])
		hostHeader := HeaderValue(reqLines[1:], "Host")
		if hostHeader != "" {
			log.Printf("[session %s] Host header: %s", s.sessionID, hostHeader)
		}
		cfIP := HeaderValue(reqLines[1:], "CF-Connecting-IP")
		if cfIP != "" {
			log.Printf("[session %s] CF-Connecting-IP header: %s", s.sessionID, cfIP)
		}
	}

	s.client.SetReadDeadline(time.Time{})

	if WebSocketHandler(s, reqLines[1:]) {
		s.Relay()
	}
}

func (s *Session) Relay() {
	defer func() {
		s.Close()
		s.server.Remove(s)
		log.Printf("[session %s] Connection closed.", s.sessionID)
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := io.Copy(s.target, s.client)
		if err != nil && !isIgnorableError(err) {
			log.Printf("[session %s] Error copying client to target: %v", s.sessionID, err)
		}
		s.target.Close()
	}()

	go func() {
		defer wg.Done()
		_, err := io.Copy(s.client, s.target)
		if err != nil && !isIgnorableError(err) {
			log.Printf("[session %s] Error copying target to client: %v", s.sessionID, err)
		}
		s.client.Close()
	}()

	wg.Wait()
}

func isIgnorableError(err error) bool {
	if err == io.EOF {
		return true
	}
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "use of closed network connection")
}
