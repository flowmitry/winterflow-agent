package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/grpc/client"
	"winterflow-agent/pkg/backoff"
	"winterflow-agent/pkg/metrics"
)

// Agent represents the application agent
type Agent struct {
	client         *client.Client
	config         *config.Config
	startTime      time.Time
	metricsFactory *metrics.MetricFactory
}

// NewAgent creates a new agent instance
func NewAgent(config *config.Config) (*Agent, error) {
	c, err := client.NewClient(config.GRPCServerAddress)
	if err != nil {
		return nil, err
	}

	start := time.Now()

	return &Agent{
		client:         c,
		config:         config,
		startTime:      start,
		metricsFactory: metrics.NewMetricFactory(start),
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
	return a.metricsFactory.Collect()
}

// StartHeartbeat starts the heartbeat stream
func (a *Agent) StartHeartbeat(accessToken string) error {
	log.Printf("Collecting system metrics for heartbeat")
	metricsProvider := a.collectMetrics

	log.Printf("Getting system capabilities for heartbeat")
	capabilities := GetSystemCapabilities().ToMap()

	log.Printf("Starting heartbeat stream with server")
	return a.client.StartHeartbeatStream(
		a.config.ServerID,
		accessToken,
		metricsProvider,
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

		// Unrecoverable errors should bubble up to abort agent run.
		if err == client.ErrUnrecoverable || err == client.ErrUnrecoverableAgentAlreadyConnected {
			return "", err
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
