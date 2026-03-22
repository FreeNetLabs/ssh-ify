package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Addr    string `json:"addr"`
	Port    int    `json:"port"`
	KeyPath string `json:"key"`
	Banner  string `json:"banner"`
	Users   []User `json:"users"`
}

type User struct {
	Name     string `json:"user"`
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
