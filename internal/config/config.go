// Package config handles loading, saving, and encrypting configuration files.
package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"

	"github.com/blackswarm/hive/internal/crypto"
)

// Config holds all tunable parameters for both agent and server modes.
type Config struct {
	Mode       string   `json:"mode"`        // "agent" or "server"
	ServerPort int      `json:"server_port"` // port for the server listener (TLS)
	Proxies    []string `json:"proxies"`     // fronting proxy domains
	AgentID    string   `json:"agent_id"`    // unique identifier for the agent
	SecretKey  []byte   `json:"-"`           // 32‑byte AES key, never serialised directly
}

// configFile is the JSON‑friendly representation of Config.
type configFile struct {
	Mode       string   `json:"mode"`
	ServerPort int      `json:"server_port"`
	Proxies    []string `json:"proxies"`
	AgentID    string   `json:"agent_id"`
	SecretKey  string   `json:"secret_key"` // base64-encoded
}

// LoadConfig reads and decrypts a configuration file.
// The key parameter must be the 32‑byte AES‑256 key used to encrypt the file.
func LoadConfig(path string, key []byte) (*Config, error) {
	if len(key) != 32 {
		return nil, errors.New("config: key must be 32 bytes")
	}

	ciphertext, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	plain, err := crypto.Decrypt(ciphertext, key)
	if err != nil {
		return nil, err
	}

	var cf configFile
	if err := json.Unmarshal(plain, &cf); err != nil {
		return nil, err
	}

	secretKey, err := base64.StdEncoding.DecodeString(cf.SecretKey)
	if err != nil {
		return nil, errors.New("config: invalid secret key encoding")
	}

	return &Config{
		Mode:       cf.Mode,
		ServerPort: cf.ServerPort,
		Proxies:    cf.Proxies,
		AgentID:    cf.AgentID,
		SecretKey:  secretKey,
	}, nil
}

// SaveConfig encrypts and writes a configuration file.
// The key parameter is the 32‑byte AES‑256 key; it must not be nil or empty.
func SaveConfig(path string, cfg *Config, key []byte) error {
	if len(key) != 32 {
		return errors.New("config: key must be 32 bytes")
	}

	cf := configFile{
		Mode:       cfg.Mode,
		ServerPort: cfg.ServerPort,
		Proxies:    cfg.Proxies,
		AgentID:    cfg.AgentID,
		SecretKey:  base64.StdEncoding.EncodeToString(cfg.SecretKey),
	}

	plain, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}

	ciphertext, err := crypto.Encrypt(plain, key)
	if err != nil {
		return err
	}

	return os.WriteFile(path, ciphertext, 0600)
}
