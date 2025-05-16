package api

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"time"
	log "winterflow-agent/pkg/log"

	"winterflow-agent/internal/config"
	"winterflow-agent/pkg/certs"
)

// RegistrationError represents a structured error response from the server
type RegistrationError struct {
	Success bool `json:"success"`
	Data    struct {
		Error string `json:"error"`
	} `json:"data"`
}

// RegisterAgent handles the agent registration process
func RegisterAgent(configPath string) error {
	// Load config to get server URL
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return log.Errorf("failed to load configuration: %v", err)
	}

	client := NewClient(cfg.APIBaseURL)

	// Try to load existing config to get server_id
	var existingServerID string
	if cfg.ServerID != "" {
		existingServerID = cfg.ServerID
		log.Printf("[DEBUG] Using existing server_id: %s", existingServerID)
	}

	// Generate agent private key
	log.Printf("[DEBUG] Generating agent private key at: %s", cfg.AgentPrivateKeyPath)
	if err := certs.GeneratePrivateKey(cfg.AgentPrivateKeyPath); err != nil {
		return log.Errorf("failed to generate agent private key: %v", err)
	}

	// Create CSR
	log.Printf("[DEBUG] Creating CSR at: %s", cfg.CSRPath)
	certificateID := uuid.New().String()
	csrData, err := certs.CreateCSR(certificateID, cfg.AgentPrivateKeyPath, cfg.CSRPath)
	if err != nil {
		return log.Errorf("failed to create CSR: %v", err)
	}

	// Request registration code and submit CSR
	resp, err := client.RequestRegistrationCode(existingServerID, certificateID, csrData)
	if err != nil {
		// Check if it's an API error
		if apiErr, ok := err.(*APIError); ok {
			if apiErr.StatusCode == 400 {
				// Parse the structured error for 400 responses
				var regErr RegistrationError
				if err := json.Unmarshal([]byte(apiErr.Body), &regErr); err == nil {
					return log.Errorf("registration failed: %s", regErr.Data.Error)
				}
			}
			// For other status codes, show a generic error
			return log.Errorf("server error: HTTP %d - please try again later", apiErr.StatusCode)
		}
		// For non-API errors (network issues, etc)
		return log.Errorf("connection error: %v", err)
	}

	// Save server_id to config immediately if it's new
	if existingServerID == "" && resp.Data.ServerID != "" {
		cfg.ServerID = resp.Data.ServerID
		if err := config.SaveConfig(cfg, configPath); err != nil {
			log.Printf("[WARN] Failed to save server_id to config: %v", err)
		} else {
			log.Printf("[DEBUG] Saved new server_id to config: %s", resp.Data.ServerID)
		}
	}

	// Save the certificate if it was returned
	if resp.Data.CertificateData != "" {
		log.Printf("[DEBUG] Saving certificate at: %s", cfg.CertificatePath)
		if err := certs.SaveCertificate(resp.Data.CertificateData, cfg.CertificatePath); err != nil {
			return log.Errorf("failed to save certificate: %v", err)
		}
	}

	// Format the code with a dash
	code := resp.Data.Code
	if len(code) == 6 {
		code = code[:3] + "-" + code[3:]
	}

	// Parse and format the expiration time
	expiresAt, err := time.Parse(time.RFC3339, resp.Data.ExpiresAt)
	if err != nil {
		// Fallback to raw string if parsing fails
		fmt.Printf("Warning: Failed to parse expiration time: %v\n", err)
	}

	// Show instructions to the user
	fmt.Println("\n=== WinterFlow.io Agent Registration ===")
	fmt.Printf("Registration Code: %s\n", code)
	if err == nil {
		fmt.Printf("Expires at: %s\n\n", expiresAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("Expires at: %s\n\n", resp.Data.ExpiresAt)
	}
	fmt.Println("To complete the registration:")
	fmt.Println("1. Go to your WinterFlow.io dashboard")
	fmt.Println("2. Click on the 'Add Server' button")
	fmt.Println("3. Enter the registration code above")
	fmt.Println("4. Wait for the registration to complete")
	fmt.Println("\nWaiting for registration confirmation...")

	// Poll for registration status
	for {
		statusResp, err := client.GetRegistrationStatus(resp.Data.ServerID, resp.Data.Token)
		if err != nil {
			// Check if it's an API error
			if apiErr, ok := err.(*APIError); ok {
				if apiErr.StatusCode == 400 {
					// For 400 errors, start a new registration
					fmt.Println("\nRegistration code has expired.")
					fmt.Println("Starting a new registration process...")
					return RegisterAgent(configPath)
				}
				// For other status codes, show a generic error
				return log.Errorf("server error: HTTP %d - please try again later", apiErr.StatusCode)
			}
			// For non-API errors
			return log.Errorf("connection error: %v", err)
		}

		switch statusResp.Data.Status {
		case "registered":
			// Update the configuration with the token
			cfg.ServerToken = resp.Data.Token
			if err := config.SaveConfig(cfg, configPath); err != nil {
				return log.Errorf("failed to save configuration: %v", err)
			}

			fmt.Println("\n=== Registration Successful ===")
			fmt.Println("The agent has been successfully registered and configured.")
			fmt.Println("\nNext steps:")
			fmt.Println("Visit the WinterFlow.io dashboard and enjoy!")
			return nil

		case "expired", "unknown":
			fmt.Println("\nRegistration code has expired or is invalid.")
			fmt.Println("Starting a new registration process...")
			return RegisterAgent(configPath)

		case "pending":
			// Wait before checking again
			time.Sleep(5 * time.Second)
			continue
		}
	}
}
