package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"winterflow-agent/internal/application/config"
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
func RegisterAgent(configPath string, orchestrator string) error {
	// Load config to get server URL
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}

	// If orchestrator specified, validate and persist it
	if orchestrator != "" {
		if err := cfg.SetOrchestrator(config.OrchestratorType(orchestrator)); err != nil {
			return fmt.Errorf("invalid orchestrator type: %v", err)
		}
		if err := config.SaveConfig(cfg, configPath); err != nil {
			return fmt.Errorf("failed to save orchestrator to config: %v", err)
		}
	}

	// Check if agent is already registered
	if cfg.AgentStatus == config.AgentStatusRegistered {
		fmt.Println("\n=== Agent Already Registered ===")
		return nil
	}

	client := NewClient(cfg.GetAPIBaseURL())

	// Try to load existing config to get agent_id
	var existingAgentID string
	if cfg.AgentID != "" {
		existingAgentID = cfg.AgentID
		fmt.Printf("Using existing agent_id: %s", existingAgentID)
	}

	// Generate agent private key
	fmt.Printf("Generating agent private key at: %s", cfg.GetPrivateKeyPath())
	if err := certs.GeneratePrivateKey(cfg.GetPrivateKeyPath()); err != nil {
		return fmt.Errorf("failed to generate agent private key: %v", err)
	}

	// Create CSR
	fmt.Printf("Creating CSR at: %s", cfg.GetCSRPath())
	certificateID := uuid.New().String()
	csrData, err := certs.CreateCSR(certificateID, cfg.GetPrivateKeyPath(), cfg.GetCSRPath())
	if err != nil {
		return fmt.Errorf("failed to create CSR: %v", err)
	}

	// Request registration code and submit CSR
	resp, err := client.RequestRegistrationCode(existingAgentID, csrData)
	if err != nil {
		// Check if it's an API error
		if apiErr, ok := err.(*APIError); ok {
			if apiErr.StatusCode == 400 {
				// Parse the structured error for 400 responses
				var regErr RegistrationError
				if err := json.Unmarshal([]byte(apiErr.Body), &regErr); err == nil {
					return fmt.Errorf("registration failed: %s", regErr.Data.Error)
				}
			}
			// For other status codes, show a generic error
			return fmt.Errorf("server error: HTTP %d - please try again later", apiErr.StatusCode)
		}
		// For non-API errors (network issues, etc)
		return fmt.Errorf("connection error: %v", err)
	}

	// Save agent_id to config immediately if it's new
	if existingAgentID == "" && resp.Data.AgentID != "" {
		cfg.AgentID = resp.Data.AgentID
		// Set agent status to pending during registration process
		cfg.AgentStatus = config.AgentStatusPending
		if err := config.SaveConfig(cfg, configPath); err != nil {
			fmt.Printf("Failed to save agent_id to config: %v", err)
		} else {
			fmt.Printf("Saved new agent_id and set status to pending in config: %s", resp.Data.AgentID)
		}
	}

	fmt.Printf("Saving certificate at: %s", cfg.GetCertificatePath())
	if err := certs.SaveCertificate(resp.Data.CertificateData, cfg.GetCertificatePath()); err != nil {
		return fmt.Errorf("failed to save certificate: %v", err)
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
		statusResp, err := client.GetRegistrationStatus(resp.Data.AgentID)
		if err != nil {
			// Check if it's an API error
			if apiErr, ok := err.(*APIError); ok {
				if apiErr.StatusCode == 400 {
					// Reset agent status to unknown before restarting registration
					cfg.AgentStatus = config.AgentStatusUnknown
					if err := config.SaveConfig(cfg, configPath); err != nil {
						fmt.Printf("Failed to reset agent status to unknown: %v", err)
					}

					// For 400 errors, start a new registration
					fmt.Println("\nRegistration code has expired.")
					fmt.Println("Starting a new registration process...")
					return RegisterAgent(configPath, orchestrator)
				}
				// For other status codes, show a generic error
				return fmt.Errorf("server error: HTTP %d - please try again later", apiErr.StatusCode)
			}
			// For non-API errors
			return fmt.Errorf("connection error: %v", err)
		}

		switch statusResp.Data.Status {
		case "registered":
			// Update agent status to registered
			cfg.AgentStatus = config.AgentStatusRegistered
			if err := config.SaveConfig(cfg, configPath); err != nil {
				fmt.Printf("Failed to update agent status to registered: %v", err)
			} else {
				fmt.Printf("Updated agent status to registered")
			}

			fmt.Println("\n=== Registration Successful ===")
			fmt.Println("The agent has been successfully registered and configured.")
			fmt.Println("\nNext steps:")
			fmt.Println("Visit the WinterFlow.io dashboard and enjoy!")
			return nil

		case "expired", "unknown":
			// Reset agent status to unknown before restarting registration
			cfg.AgentStatus = config.AgentStatusUnknown
			if err := config.SaveConfig(cfg, configPath); err != nil {
				fmt.Printf("Failed to reset agent status to unknown: %v", err)
			}

			fmt.Println("\nRegistration code has expired or is invalid.")
			fmt.Println("Starting a new registration process...")
			return RegisterAgent(configPath, orchestrator)

		case "pending":
			// Wait before checking again
			time.Sleep(5 * time.Second)
			continue
		}
	}
}
