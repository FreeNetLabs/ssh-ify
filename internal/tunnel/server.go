package tunnel

import (
	"fmt"
	"log"
	"net"

	"github.com/FreeNetLabs/ssh-ify/internal/config"
	"github.com/FreeNetLabs/ssh-ify/internal/ssh"
)

type Server struct {
	host      string
	port      int
	sshConfig *ssh.ServerConfig
}

func Start(cfg *config.Config) {
	s := NewServer(cfg)
	s.Run()
}

func NewServer(cfg *config.Config) *Server {
	sshCfg, err := ssh.NewConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize SSH config: %v", err)
	}

	return &Server{
		host:      cfg.Addr,
		port:      cfg.Port,
		sshConfig: sshCfg,
	}
}

func (s *Server) Run() {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on TCP %s: %v", addr, err)
	}
	log.Printf("TCP server listening on %s", addr)

	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		session := &Session{
			client:    conn,
			sshConfig: s.sshConfig,
		}
		go session.Serve()
	}
}
