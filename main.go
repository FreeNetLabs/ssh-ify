package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ayanrajpoot10/ssh-ify/internal/config"
	"github.com/ayanrajpoot10/ssh-ify/internal/ssh"
	"github.com/ayanrajpoot10/ssh-ify/internal/tunnel"
)

func main() {
	cfg, err := config.LoadConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read config: %v\n", err)
		os.Exit(1)
	}

	if err := ssh.InitializeAuth(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to configure authentication: %v\n", err)
		os.Exit(1)
	}

	sshCfg, err := ssh.NewConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize SSH server config: %v\n", err)
		os.Exit(1)
	}

	log.Println("Starting ssh-ify tunnel server...")
	tunnel.SetSSHConfig(sshCfg)

	tunnel.StartServer(cfg)
}
