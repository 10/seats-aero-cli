package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type fileConfig struct {
	APIKey string `json:"api_key"`
}

func Path() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Abs(filepath.Join(base, "seatsaero", "config.json"))
}

func Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", err
	}
	return cfg.APIKey, nil
}

func Write(path, key string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	// Tighten existing files before writing any secret bytes.
	if err := f.Chmod(0600); err != nil {
		return err
	}
	if err := f.Truncate(0); err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(fileConfig{APIKey: key}); err != nil {
		return err
	}
	return f.Close()
}
