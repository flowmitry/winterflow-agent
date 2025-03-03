package registration

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"winterflow-agent/internal/config"
	internalsystem "winterflow-agent/internal/system"
	"winterflow-agent/internal/winterflow/client"
)

const (
	// RegistrationURL is the URL where users should enter their code
	RegistrationURL = "https://winterflow.io/activate"
	// PollInterval is the interval between polling for registration status
	PollInterval = 5 * time.Second
	// MaxPollDuration is the maximum time to wait for registration
	MaxPollDuration = 5 * time.Minute
)

// DeviceCodeRequest represents the initial registration request
type DeviceCodeRequest struct {
	DeviceID string `json:"device_id"`
	Hostname string `json:"hostname"`
}

// DeviceCodeResponse represents the response with the code to show to the user
type DeviceCodeResponse struct {
	UserCode        string `json:"user_code"`
	DeviceCode      string `json:"device_code"`
	ExpiresIn       int    `json:"expires_in"`       // Seconds until the code expires
	Interval        int    `json:"interval"`         // Polling interval in seconds
	VerificationURL string `json:"verification_url"` // URL where user should enter the code
}

// RegistrationStatusRequest represents the registration status check request
type RegistrationStatusRequest struct {
	DeviceCode string `json:"device_code"`
}

// RegistrationStatusResponse represents the registration status check response
type RegistrationStatusResponse struct {
	Status     string `json:"status"`      // "pending" or "complete"
	AgentToken string `json:"agent_token"` // Only present when status is "complete"
}

// Register starts the device flow registration process
func Register(configPath string) error {
	// Create config manager
	configManager := config.NewManager(configPath)

	// Check if already registered
	isRegistered, err := configManager.IsRegistered()
	if err != nil {
		return fmt.Errorf("failed to check registration status: %w", err)
	}
	if isRegistered {
		return fmt.Errorf("agent is already registered")
	}

	// Read device ID from system for registration
	deviceID, err := internalsystem.GetDeviceID()
	if err != nil {
		return fmt.Errorf("failed to read device ID: %w", err)
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	// Create API client with real device ID for registration
	apiClient, err := client.NewClient("", deviceID) // No credentials needed for initial request
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Request device code
	reqBody := DeviceCodeRequest{
		DeviceID: deviceID,
		Hostname: hostname,
	}

	respData, err := apiClient.DoRequest("POST", "/agents/device-code", reqBody)
	if err != nil {
		return fmt.Errorf("failed to get device code: %w", err)
	}

	var deviceResp DeviceCodeResponse
	if err := json.Unmarshal(respData, &deviceResp); err != nil {
		return fmt.Errorf("failed to parse device code response: %w", err)
	}

	// Show instructions to user
	fmt.Printf("\nTo register this agent, please visit:\n\n")
	fmt.Printf("    %s\n\n", deviceResp.VerificationURL)
	fmt.Printf("And enter the following code:\n\n")
	fmt.Printf("    %s\n\n", deviceResp.UserCode)
	fmt.Printf("Waiting for registration to complete...\n")

	// Poll for registration status
	pollInterval := time.Duration(deviceResp.Interval) * time.Second
	if pollInterval < PollInterval {
		pollInterval = PollInterval
	}

	deadline := time.Now().Add(MaxPollDuration)
	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)

		statusReq := RegistrationStatusRequest{
			DeviceCode: deviceResp.DeviceCode,
		}

		respData, err := apiClient.DoRequest("POST", "/agents/registration-status", statusReq)
		if err != nil {
			continue // Just retry on error
		}

		var statusResp RegistrationStatusResponse
		if err := json.Unmarshal(respData, &statusResp); err != nil {
			continue // Just retry on parse error
		}

		if statusResp.Status == "complete" {
			cfg := &config.Config{
				DeviceID:   deviceID,
				AgentToken: statusResp.AgentToken,
			}

			if err := configManager.SaveConfig(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("registration timed out - please try again")
}
