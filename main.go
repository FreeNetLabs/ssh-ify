package main

import (
	"log"

	"github.com/ayanrajpoot10/ssh-ify/internal/config"
	"github.com/ayanrajpoot10/ssh-ify/internal/tunnel"
)

func main() {
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	tunnel.StartServer(cfg)
}
