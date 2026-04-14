package main

import (
	"log"

	"github.com/spf13/pflag"

	"github.com/FreeNetLabs/ssh-ify/internal/config"
	"github.com/FreeNetLabs/ssh-ify/internal/proxy"
	"github.com/FreeNetLabs/ssh-ify/internal/ssh"
)

func main() {
	log.SetFlags(log.Ltime)

	configPath := pflag.StringP("config", "c", "config.json", "Path to config file")

	pflag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config err: %v", err)
	}

	sshServer, err := ssh.NewServer(cfg)
	if err != nil {
		log.Fatalf("ssh server err: %v", err)
	}

	proxy.Start(cfg, sshServer)
}
