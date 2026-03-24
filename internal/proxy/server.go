package proxy

import (
	"fmt"
	"log"
	"net"

	"github.com/FreeNetLabs/ssh-ify/internal/config"
	"github.com/FreeNetLabs/ssh-ify/internal/ssh"
)

type Server struct {
	cfg       *config.Config
	sshServer *ssh.Server
}

func Start(cfg *config.Config, sshServer *ssh.Server) {
	s := &Server{
		cfg:       cfg,
		sshServer: sshServer,
	}
	s.Run()
}

func (s *Server) Run() {
	addr := fmt.Sprintf("%s:%d", s.cfg.Addr, s.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s err: %v", addr, err)
	}
	defer ln.Close()

	log.Printf("Server listening on %s", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept err: %v", err)
			continue
		}

		clientConn := &Conn{
			client:    conn,
			sshServer: s.sshServer,
		}
		go clientConn.Serve()
	}
}
