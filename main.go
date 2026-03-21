package main

import (
	"log"

	"github.com/ayanrajpoot10/ssh-ify/internal/config"
	"github.com/ayanrajpoot10/ssh-ify/internal/ssh"
	"github.com/ayanrajpoot10/ssh-ify/internal/tunnel"
)

func main() {
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	sshCfg, err := ssh.NewConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize SSH server config: %v", err)
	}

	tunnel.SetSSHConfig(sshCfg)

	tunnel.StartServer(cfg)
}
