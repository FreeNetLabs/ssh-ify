package main

import (
	"flag"
	"log"

	"github.com/FreeNetLabs/ssh-ify/internal/config"
	"github.com/FreeNetLabs/ssh-ify/internal/proxy"
)

func main() {
	configPath := flag.String("config", "config.json", "path to configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	proxy.Start(cfg)
}
