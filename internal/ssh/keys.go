package ssh

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

func loadHostKey() (ssh.Signer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir err: %w", err)
	}

	keyPath := filepath.Join(home, ".ssh", "id_rsa")
	privateBytes, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("missing host key: run ssh-keygen -f %s", keyPath)
		}
		return nil, fmt.Errorf("read host key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, fmt.Errorf("parse host key: %w", err)
	}

	return signer, nil
}
