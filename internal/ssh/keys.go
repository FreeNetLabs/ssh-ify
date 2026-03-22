package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

const HostKeyPath = "/etc/ssh-ify/host_key"

func LoadHostKey() (ssh.Signer, error) {
	privateBytes, err := os.ReadFile(HostKeyPath)
	if err != nil {
		if err := os.MkdirAll("/etc/ssh-ify", 0700); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %v", err)
		}
		if err := GenerateHostKey(HostKeyPath); err != nil {
			return nil, fmt.Errorf("failed to generate host key: %v", err)
		}
		privateBytes, err = os.ReadFile(HostKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read generated host key: %v", err)
		}
	}

	return ssh.ParsePrivateKey(privateBytes)
}

func GenerateHostKey(keyPath string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	privDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	}
	privateBytes := pem.EncodeToMemory(privBlock)

	return os.WriteFile(keyPath, privateBytes, 0600)
}
