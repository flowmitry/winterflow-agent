package model

import (
	"encoding/json"
)

// AppConfig represents the configuration of an app
type AppConfig struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Files     []AppFile     `json:"files"`
	Variables []AppVariable `json:"variables"`
}

// AppFile represents a file in the app configuration
type AppFile struct {
	ID          string `json:"id"`
	IsEncrypted bool   `json:"is_encrypted"`
	Filename    string `json:"filename"`
	Origin      string `json:"origin"`
}

// AppVariable represents a variable in the app configuration
type AppVariable struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IsEncrypted bool   `json:"is_encrypted"`
	Origin      string `json:"origin"`
	Type        string `json:"type"`
}

// ParseAppConfig parses the app configuration from JSON bytes
func ParseAppConfig(configBytes []byte) (*AppConfig, error) {
	var config AppConfig
	err := json.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
