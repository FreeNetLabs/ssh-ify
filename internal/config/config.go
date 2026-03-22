package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	ListenAddress  string     `json:"addr"`
	ListenPort     int        `json:"port"`
	SSHHostKeyPath string     `json:"key"`
	Banner         string     `json:"banner"`
	Users          []AuthUser `json:"users"`
}

type AuthUser struct {
	Username string `json:"user"`
	Password string `json:"pass"`
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
