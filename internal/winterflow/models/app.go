package models

import (
	"encoding/json"
	"fmt"
)

type AppType string

const (
	AppTypeDockerCompose AppType = "docker_compose"
	AppTypeDockerSwarm   AppType = "docker_swarm"
)

// String returns the string representation of AppType
func (at AppType) String() string {
	return string(at)
}

// IsValid checks if the AppType value is valid
func (at AppType) IsValid() bool {
	switch at {
	case AppTypeDockerCompose, AppTypeDockerSwarm:
		return true
	}
	return false
}

// UnmarshalJSON implements the json.Unmarshaler interface for AppType
func (at *AppType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	temp := AppType(s)
	if !temp.IsValid() {
		return fmt.Errorf("invalid AppType: %s", s)
	}

	*at = temp
	return nil
}

// MarshalJSON implements the json.Marshaler interface for AppType
func (at AppType) MarshalJSON() ([]byte, error) {
	return json.Marshal(at.String())
}

// AppConfig represents the configuration of an app
type AppConfig struct {
	ID           string           `json:"id"`
	Type         AppType          `json:"type"`
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
