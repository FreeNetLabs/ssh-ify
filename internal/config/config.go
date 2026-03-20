package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	var configDir string

	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		configDir = filepath.Join(xdgConfig, "ssh-ify")
	} else if appData := os.Getenv("APPDATA"); appData != "" {
		configDir = filepath.Join(appData, "ssh-ify")
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		configDir = filepath.Join(homeDir, ".config", "ssh-ify")
	} else {
		return "", err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}

func GetConfigFilePath() (string, error) {
	cwd, err := os.Getwd()
	if err == nil {
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
		path, err = GetConfigFilePath()
		if err != nil {
			return nil, err
		}
	}

	cfg := &Config{
		ListenAddress:  DefaultListenAddress,
		ListenPort:     DefaultListenPort,
		SSHHostKeyPath: DefaultSSHHostKeyPath,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			loadUsersFromEnv(cfg)
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.ListenAddress == "" {
		cfg.ListenAddress = DefaultListenAddress
	}
	if cfg.ListenPort == 0 {
		cfg.ListenPort = DefaultListenPort
	}
	if cfg.SSHHostKeyPath == "" {
		cfg.SSHHostKeyPath = DefaultSSHHostKeyPath
	}

	loadUsersFromEnv(cfg)

	return cfg, nil
}

func loadUsersFromEnv(cfg *Config) {
	if cfg == nil {
		return
	}

	usersFromEnv := os.Getenv("SSH_IFY_USERS")
	if usersFromEnv == "" {
		return
	}

	pairs := strings.Split(usersFromEnv, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}
		cfg.Users = append(cfg.Users, AuthUser{Username: strings.TrimSpace(parts[0]), Password: strings.TrimSpace(parts[1])})
	}
}
