package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	ListenAddress  string     `json:"listen_address"`
	ListenPort     int        `json:"listen_port"`
	SSHHostKeyPath string     `json:"ssh_host_key_path"`
	Banner         string     `json:"banner"`
	Users          []AuthUser `json:"users"`
}

type AuthUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	return &cfg, err
}
