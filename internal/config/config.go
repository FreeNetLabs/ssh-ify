package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	ListenAddress  string     `json:"listen_address"`
	ListenPort     int        `json:"listen_port"`
	SSHHostKeyPath string     `json:"ssh_host_key_path"`
	Users          []AuthUser `json:"users"`
}

type AuthUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

const (
	DefaultListenAddress  = "0.0.0.0"
	DefaultListenPort     = 80
	DefaultSSHHostKeyPath = "host_key"
)

func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	appConfigDir := filepath.Join(configDir, "ssh-ify")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return appConfigDir, nil
}

func GetConfigFilePath() (string, error) {
	if cwd, err := os.Getwd(); err == nil {
		localPath := filepath.Join(cwd, "config.json")
		if _, err := os.Stat(localPath); err == nil {
			return localPath, nil
		}
	}

	const etcPath = "/etc/ssh-ify/config.json"
	if _, err := os.Stat(etcPath); err == nil {
		return etcPath, nil
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

func LoadConfig(path string) (*Config, error) {
	if path == "" {
		var err error
		if path, err = GetConfigFilePath(); err != nil {
			return nil, fmt.Errorf("could not determine config file path: %w", err)
		}
	}

	cfg := &Config{
		ListenAddress:  DefaultListenAddress,
		ListenPort:     DefaultListenPort,
		SSHHostKeyPath: DefaultSSHHostKeyPath,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", path, err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file '%s': %w", path, err)
	}

	return cfg, nil
}
