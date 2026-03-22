package proxy

import (
	"fmt"
	"log"
	"net"

	"github.com/FreeNetLabs/ssh-ify/internal/config"
	"github.com/FreeNetLabs/ssh-ify/internal/ssh"
)

type Server struct {
	cfg    *config.Config
	sshCfg *ssh.ServerConfig
}

func Start(cfg *config.Config) {
	s := &Server{
		cfg: cfg,
	}
	s.Run()
}

func (s *Server) Run() {
	addr := fmt.Sprintf("%s:%d", s.cfg.Addr, s.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on TCP %s: %v", addr, err)
	}
	defer ln.Close()

	log.Printf("Server listening on %s", addr)

	sshCfg, err := ssh.NewConfig(s.cfg)
	if err != nil {
		log.Fatalf("Failed to initialize SSH config: %v", err)
	}
	s.sshCfg = sshCfg

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		tunnelConn := &Conn{
			client: conn,
			sshCfg: s.sshCfg,
		}
		go tunnelConn.Serve()
	}
}
