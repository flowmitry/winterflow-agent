package device

import (
	"fmt"
	"os"
	"strings"
)

const (
	// MachineIDFile is the path to the machine ID file
	MachineIDFile = "/etc/machine-id"
)

// GetDeviceID reads and returns the current device ID from the machine ID file
func GetDeviceID() (string, error) {
	data, err := os.ReadFile(MachineIDFile)
	if err != nil {
		return "", fmt.Errorf("failed to read machine ID file: %w", err)
	}

	// Clean up the machine ID (remove whitespace and newlines)
	deviceID := strings.TrimSpace(string(data))
	if deviceID == "" {
		return "", fmt.Errorf("device ID is empty")
	}

	return deviceID, nil
}
