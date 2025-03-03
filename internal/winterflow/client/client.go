package client

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// APIEndpoint is the base URL for the Winterflow API
	APIEndpoint = "https://api.winterflow.io/v1"
)

// Client represents a Winterflow API client
type Client struct {
	agentToken string
	machineID  string
	baseURL    string
}

// NewClient creates a new Winterflow API client
func NewClient(agentToken string, machineID string) (*Client, error) {
	if machineID == "" {
		return nil, fmt.Errorf("machine ID is required")
	}

	return &Client{
		agentToken: agentToken,
		machineID:  machineID,
		baseURL:    APIEndpoint,
	}, nil
}

// signRequest signs the request with HMAC-SHA256
func (c *Client) signRequest(method, path string, timestamp string, body []byte) string {
	// Create string to sign:
	// METHOD + "\n" + PATH + "\n" + TIMESTAMP + "\n" + MACHINE_ID + "\n" + BODY_HASH
	bodyHash := sha256Hash(body)
	stringToSign := strings.Join([]string{
		method,
		path,
		timestamp,
		c.machineID,
		bodyHash,
	}, "\n")

	// Create HMAC-SHA256 signature
	h := hmac.New(sha256.New, []byte(c.agentToken))
	h.Write([]byte(stringToSign))
	signature := hex.EncodeToString(h.Sum(nil))

	return signature
}

// sha256Hash calculates SHA256 hash of the data
func sha256Hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// DoRequest performs an HTTP request with proper signing
func (c *Client) DoRequest(method, path string, body interface{}) ([]byte, error) {
	var bodyData []byte
	var err error

	if body != nil {
		bodyData, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Create request
	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	timestamp := time.Now().UTC().Format(time.RFC3339)
	signature := c.signRequest(method, path, timestamp, bodyData)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Winterflow-Timestamp", timestamp)
	req.Header.Set("X-Winterflow-Signature", signature)
	req.Header.Set("X-Winterflow-Machine-ID", c.machineID)

	if c.agentToken != "" {
		req.Header.Set("X-Winterflow-Agent-Token", c.agentToken)
	}

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}
