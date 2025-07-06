package model

import (
	"encoding/json"
	"fmt"
)

// AppConfig represents the configuration of an app
type AppConfig struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Icon      string        `json:"icon"`
	Color     string        `json:"color"`
	Files     []AppFile     `json:"files"`
	Variables []AppVariable `json:"variables"`
}

// AppFile represents a file in the app configuration
type AppFile struct {
	ID          string      `json:"id"`
	Filename    string      `json:"filename"`
	IsEncrypted bool        `json:"is_encrypted"`
	Type        ContentType `json:"type"`
}

// AppVariable represents a variable in the app configuration
type AppVariable struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	IsEncrypted bool        `json:"is_encrypted"`
	Type        ContentType `json:"type"`
}

type ContentType string

const (
	ContentTypeTemplate ContentType = "template"
	ContentTypeUser     ContentType = "user"
	ContentTypeExpose   ContentType = "expose"
)

func (ct ContentType) Validate() error {
	switch ct {
	case ContentTypeTemplate, ContentTypeUser, ContentTypeExpose:
		return nil
	default:
		return fmt.Errorf("invalid content type: %s", ct)
	}
}

func (ct ContentType) Value() string {
	return string(ct)
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
