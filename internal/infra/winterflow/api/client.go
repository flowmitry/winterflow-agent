package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
	"winterflow-agent/pkg/log"
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
	AgentID       string `json:"agent_id"`
	CertificateID string `json:"certificate_id"`
	CSRData       string `json:"csr_data"`
}

// RegistrationResponse represents the response from registration API
type RegistrationResponse struct {
	Success bool `json:"success"`
	Data    struct {
		AgentID         string `json:"agent_id"`
		Code            string `json:"code"`
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
func (c *Client) RequestRegistrationCode(agentID string, csrData string) (*RegistrationResponse, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %v", err)
	}

	// Base64 encode the CSR data before sending
	encodedCSRData := base64.StdEncoding.EncodeToString([]byte(csrData))

	reqBody := RegistrationRequest{
		Hostname: hostname,
		AgentID:  agentID,
		CSRData:  encodedCSRData,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/agents/request-registration-code", c.baseURL)
	log.Debug("Sending registration request", "url", url)
	log.Debug("Request body", "body", string(jsonData))

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

	log.Debug("Response status", "status_code", resp.StatusCode)
	log.Debug("Response body", "body", string(body))

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

	// Decode the certificate data from base64 if it exists
	if response.Data.CertificateData != "" {
		decodedCertData, err := base64.StdEncoding.DecodeString(response.Data.CertificateData)
		if err != nil {
			return nil, fmt.Errorf("failed to decode certificate data: %v", err)
		}
		response.Data.CertificateData = string(decodedCertData)
	}

	return &response, nil
}

// GetRegistrationStatus checks the registration status
func (c *Client) GetRegistrationStatus(agentID string) (*RegistrationStatusResponse, error) {
	url := fmt.Sprintf("%s/api/v1/agents/get-registration-status?agent_id=%s", c.baseURL, agentID)
	log.Debug("Checking registration status", "url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

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

	log.Debug("Response status", "status_code", resp.StatusCode)
	log.Debug("Response body", "body", string(body))

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

// GetIPAddress gets the IP address of the agent
func (c *Client) GetIPAddress() (string, error) {
	url := fmt.Sprintf("%s/api/v1/agents/get-ip", c.baseURL)
	log.Debug("Getting IP address", "url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	log.Debug("Response status", "status_code", resp.StatusCode)
	log.Debug("Response body", "body", string(body))

	if resp.StatusCode != http.StatusOK {
		return "", &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	return string(body), nil
}
