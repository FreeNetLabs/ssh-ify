package tunnel

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/FreeNetLabs/ssh-ify/internal/config"
	"github.com/FreeNetLabs/ssh-ify/internal/ssh"
)

type Server struct {
	host        string
	port        int
	ctx         context.Context
	cancel      context.CancelFunc
	conns       sync.Map
	activeCount int32
	wg          sync.WaitGroup
	sshConfig   *ssh.ServerConfig
}

func Start(cfg *config.Config) {
	s := NewServer(cfg)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	s.ListenAndServe()

	<-c
	s.cancel()
	s.Shutdown()
	log.Println("Shutting down...")
}

func NewServer(cfg *config.Config) *Server {
	sshCfg, err := ssh.NewConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize SSH config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		host:      cfg.ListenAddress,
		port:      cfg.ListenPort,
		ctx:       ctx,
		cancel:    cancel,
		sshConfig: sshCfg,
	}
}

func (s *Server) ListenAndServe() {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
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
