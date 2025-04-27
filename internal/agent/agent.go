package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/grpc/client"
	"winterflow-agent/pkg/backoff"
)

// Agent represents the application agent
type Agent struct {
	client    *client.Client
	config    *config.Config
	startTime time.Time
}

// NewAgent creates a new agent instance
func NewAgent(config *config.Config) (*Agent, error) {
	c, err := client.NewClient(config.GRPCServerAddress)
	if err != nil {
		return nil, err
	}

	return &Agent{
		client:    c,
		config:    config,
		startTime: time.Now(),
	}, nil
}

// Register registers the agent with the server
func (a *Agent) Register() (string, error) {
	log.Printf("Getting system capabilities")
	capabilities := GetSystemCapabilities().ToMap()

	log.Printf("Registering agent with server")
	resp, err := a.client.RegisterAgent(GetVersion(), capabilities, a.config.Features, a.config.ServerID, a.config.ServerToken)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", fmt.Errorf("registration failed: %s", resp.Message)
	}

	// Store the access token in the client
	a.client.SetAccessToken(resp.AccessToken)

	log.Printf("Agent registered successfully. Access token: %s", resp.AccessToken)
	return resp.AccessToken, nil
}

// collectMetrics collects system metrics for heartbeat
func (a *Agent) collectMetrics() map[string]string {
	// TODO: Implement actual system metrics collection
	// For now, return static metrics
	return map[string]string{
		"cpu_usage": "0.5",
		"memory":    "512MB",
		"uptime":    time.Since(a.startTime).String(),
	}
}

// StartHeartbeat starts the heartbeat stream
func (a *Agent) StartHeartbeat(accessToken string) error {
	log.Printf("Collecting system metrics for heartbeat")
	metrics := a.collectMetrics()

	log.Printf("Getting system capabilities for heartbeat")
	capabilities := GetSystemCapabilities().ToMap()

	log.Printf("Starting heartbeat stream with server")
	return a.client.StartHeartbeatStream(
		a.config.ServerID,
		accessToken,
		metrics,
		GetVersion(),
		capabilities,
		a.config.Features,
		a.config.ServerToken,
	)
}

// Close closes the agent's client connection
func (a *Agent) Close() {
	if a.client != nil {
		a.client.Close()
	}
}

// RegisterWithRetry attempts to register the agent with retry logic
func (a *Agent) RegisterWithRetry(ctx context.Context) (string, error) {
	b := backoff.New(2*time.Second, 1*time.Minute)

	for {
		token, err := a.Register()
		if err == nil {
			b.Reset()
			return token, nil
		}

		delay := b.Next()
		log.Printf("Registration failed: %v. Retrying in %s", err, delay)

		select {
		case <-time.After(delay):
			continue
		case <-ctx.Done():
			return "", fmt.Errorf("registration cancelled: %v", ctx.Err())
		}
	}
}

// Run starts the agent's main loop
func (a *Agent) Run(ctx context.Context) error {
	// Register the agent
	log.Printf("Registering agent with server: %s", a.config.GRPCServerAddress)
	accessToken, err := a.RegisterWithRetry(ctx)
	if err != nil {
		return fmt.Errorf("failed to register agent: %v", err)
	}
	log.Printf("Agent registered successfully with access token: %s", accessToken)

	// Start heartbeat stream
	log.Printf("Starting heartbeat stream")
	if err := a.StartHeartbeat(accessToken); err != nil {
		return fmt.Errorf("failed to start heartbeat stream: %v", err)
	}
	log.Printf("Heartbeat stream started successfully")

	return nil
}
