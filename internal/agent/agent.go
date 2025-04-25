package agent

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"winterflow-agent/internal/grpc/client"
	"winterflow-agent/pkg/version"
)

// Agent represents the application agent
type Agent struct {
	client *client.Client
	config *Config
}

// NewAgent creates a new agent instance
func NewAgent(config *Config) (*Agent, error) {
	c, err := client.NewClient(config.ServerAddress)
	if err != nil {
		return nil, err
	}

	return &Agent{
		client: c,
		config: config,
	}, nil
}

// Register registers the agent with the server
func (a *Agent) Register() (string, error) {
	capabilities := map[string]string{
		"os":      "darwin",
		"version": version.GetVersion(),
	}

	resp, err := a.client.RegisterAgent(version.GetVersion(), capabilities, a.config.ServerID, a.config.ServerToken)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", fmt.Errorf("registration failed: %s", resp.Message)
	}

	log.Printf("Agent registered successfully. Access token: %s", resp.AccessToken)
	return resp.AccessToken, nil
}

// StartHeartbeat starts the heartbeat stream
func (a *Agent) StartHeartbeat(accessToken string) error {
	metrics := map[string]string{
		"cpu_usage": "0.5",
		"memory":    "512MB",
	}

	return a.client.StartHeartbeatStream(a.config.ServerID, accessToken, metrics)
}

// Unregister unregisters the agent from the server
func (a *Agent) Unregister(accessToken string) error {
	resp, err := a.client.UnregisterAgent(a.config.ServerID, accessToken)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("unregistration failed: %s", resp.Message)
	}

	log.Println("Agent unregistered successfully")
	return nil
}

// Close closes the agent's client connection
func (a *Agent) Close() {
	if a.client != nil {
		a.client.Close()
	}
}

// WaitForSignal waits for an interrupt signal
func (a *Agent) WaitForSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
