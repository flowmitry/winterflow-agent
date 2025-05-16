package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
	log "winterflow-agent/pkg/log"
)

// Client represents an HTTP client for registration API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new HTTP client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// APIError represents a structured error response from the server
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	if e.StatusCode == http.StatusBadRequest {
		return fmt.Sprintf("bad_request:%s", e.Body)
	}
	return fmt.Sprintf("request_failed:%d", e.StatusCode)
}

// RegistrationRequest represents the request body for registration
type RegistrationRequest struct {
	Hostname      string `json:"hostname"`
	ServerID      string `json:"server_id"`
	CertificateID string `json:"certificate_id"`
	CSRData       string `json:"csr_data"`
}

// RegistrationResponse represents the response from registration API
type RegistrationResponse struct {
	Success bool `json:"success"`
	Data    struct {
		ServerID        string `json:"server_id"`
		Code            string `json:"code"`
		Token           string `json:"token"`
		ExpiresAt       string `json:"expires_at"`
		CertificateData string `json:"certificate_data"`
	} `json:"data"`
}

// RegistrationStatusResponse represents the response from registration status API
type RegistrationStatusResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Status string `json:"status"`
	} `json:"data"`
}

// RequestRegistrationCode requests a registration code from the server
// If csrData and commonName are provided, it also submits a CSR and receives a signed certificate
func (c *Client) RequestRegistrationCode(serverID string, csrData string, certificateID string) (*RegistrationResponse, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %v", err)
	}

	reqBody := RegistrationRequest{
		Hostname:      hostname,
		ServerID:      serverID,
		CertificateID: certificateID,
		CSRData:       csrData,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/servers/request-registration-code", c.baseURL)
	log.Printf("[DEBUG] Sending registration request to: %s", url)
	log.Printf("[DEBUG] Request body: %s", string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	log.Printf("[DEBUG] Response status: %d", resp.StatusCode)
	log.Printf("[DEBUG] Response body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	var response RegistrationResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v, body: %s", err, string(body))
	}

	return &response, nil
}

// GetRegistrationStatus checks the registration status
func (c *Client) GetRegistrationStatus(serverID, serverToken string) (*RegistrationStatusResponse, error) {
	url := fmt.Sprintf("%s/api/v1/servers/get-registration-status?server_id=%s", c.baseURL, serverID)
	log.Printf("[DEBUG] Checking registration status at: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("X-Server-Token", serverToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	log.Printf("[DEBUG] Response status: %d", resp.StatusCode)
	log.Printf("[DEBUG] Response body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	var response RegistrationStatusResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v, body: %s", err, string(body))
	}

	return &response, nil
}
