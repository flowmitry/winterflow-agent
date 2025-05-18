package models

import (
	"encoding/json"
)

// AppConfig represents the configuration of an app
type AppConfig struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Files        []AppFile        `json:"files"`
	Variables    []AppVariable    `json:"variables"`
	Networks     []AppNetwork     `json:"networks"`
	Schedules    []AppSchedule    `json:"schedules"`
	PortMappings []AppPortMapping `json:"port_mappings"`
}

// AppFile represents a file in the app configuration
type AppFile struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Origin   string `json:"origin"`
}

// AppVariable represents a variable in the app configuration
type AppVariable struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Add other fields as needed
}

// AppNetwork represents a network in the app configuration
type AppNetwork struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Add other fields as needed
}

// AppSchedule represents a schedule in the app configuration
type AppSchedule struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Add other fields as needed
}

// AppPortMapping represents a port mapping in the app configuration
type AppPortMapping struct {
	ID       string `json:"id"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	// Add other fields as needed
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
