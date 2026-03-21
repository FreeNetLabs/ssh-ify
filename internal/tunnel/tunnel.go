package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ayanrajpoot10/ssh-ify/internal/config"
	"github.com/ayanrajpoot10/ssh-ify/internal/ssh"
)

const (
	ClientReadTimeout = 60 * time.Second
	MaxHeaderSize     = 16384

	WebSocketUpgradeResponse = "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n\r\n"
)

type Server struct {
	host        string
	tcpPort     int
	ctx         context.Context
	cancel      context.CancelFunc
	conns       sync.Map
	activeCount int32
	wg          sync.WaitGroup
	sshConfig   *ssh.ServerConfig
}

type Session struct {
	client    net.Conn
	target    net.Conn
	server    *Server
	sshConfig *ssh.ServerConfig
	sessionID string
}

func (s *Server) Add(conn *Session) {
	select {
	case <-s.ctx.Done():
		return
	default:
		s.conns.Store(conn, struct{}{})
		s.wg.Add(1)
		newCount := atomic.AddInt32(&s.activeCount, 1)
		log.Println("Connection added. Active:", newCount)
	}
}

func (s *Server) Remove(conn *Session) {
	s.conns.Delete(conn)
	s.wg.Done()
	newCount := atomic.AddInt32(&s.activeCount, -1)
	log.Println("Connection removed. Active:", newCount)
}

func (s *Server) Shutdown() {
	log.Println("Closing all active connections...")
	s.conns.Range(func(key, value any) bool {
		if sess, ok := key.(*Session); ok {
			sess.Close()
		}
		return true
	})
	s.wg.Wait()
	log.Println("All sessions closed.")
}

func NewServer(cfg *config.Config) *Server {
	sshCfg, err := ssh.NewConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize SSH config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		host:      cfg.ListenAddress,
		tcpPort:   cfg.ListenPort,
		ctx:       ctx,
		cancel:    cancel,
		conns:     sync.Map{},
		sshConfig: sshCfg,
	}
}

func StartServer(cfg *config.Config) {
	s := NewServer(cfg)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	s.ListenAndServe()

	<-c
	s.cancel()
	s.Shutdown()
	log.Println("Shutting down...")
}

func (s *Server) ListenAndServe() {
	addr := fmt.Sprintf("%s:%d", s.host, s.tcpPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on TCP %s: %v", addr, err)
	}
	log.Printf("TCP server listening on %s", addr)

	go func() {
		defer ln.Close()
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				if tcpLn, ok := ln.(*net.TCPListener); ok {
					tcpLn.SetDeadline(time.Now().Add(2 * time.Second))
				}
				conn, err := ln.Accept()
				if err != nil {
					if ne, ok := err.(net.Error); ok && ne.Timeout() {
						continue
					}
					return
				}
				sess := &Session{client: conn, server: s, sessionID: conn.RemoteAddr().String(), sshConfig: s.sshConfig}
				go sess.Handle()
			}
		}
	}()
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
		s.Close()          // Clean up both connections
		s.server.Remove(s) // Remove from active map
		log.Printf("[session %s] Connection closed.", s.sessionID)
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	// Copy client → target
	go func() {
		defer wg.Done()
		_, err := io.Copy(s.target, s.client)
		if err != nil && !isIgnorableError(err) {
			log.Printf("[session %s] Error copying client to target: %v", s.sessionID, err)
		}
		s.target.Close()
	}()

	// Copy target → client
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
