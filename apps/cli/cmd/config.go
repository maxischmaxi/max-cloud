package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Credentials enthält die gespeicherten CLI-Zugangsdaten.
type Credentials struct {
	APIURL string `yaml:"api_url,omitempty"`
	APIKey string `yaml:"api_key"`
}

// configDir gibt das Konfigurationsverzeichnis zurück (~/.config/maxcloud/).
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".config", "maxcloud"), nil
}

// credentialsPath gibt den Pfad zur credentials-Datei zurück.
func credentialsPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials"), nil
}

// loadCredentials liest die gespeicherten Credentials.
func loadCredentials() (*Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading credentials: %w", err)
	}

	var creds Credentials
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}
	return &creds, nil
}

// saveCredentials speichert Credentials in ~/.config/maxcloud/credentials.
func saveCredentials(creds *Credentials) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}

	path := filepath.Join(dir, "credentials")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing credentials: %w", err)
	}

	return nil
}
