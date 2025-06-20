package agent

import (
	"context"
	"time"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/infra/ansible"
	log "winterflow-agent/pkg/log"

	"winterflow-agent/internal/config"
	"winterflow-agent/internal/infra/winterflow/grpc/client"
	"winterflow-agent/pkg/metrics"
)

// Agent represents the application agent
type Agent struct {
	client            *client.Client
	config            *config.Config
	startTime         time.Time
	metricsFactory    *metrics.MetricFactory
	systemInfoFactory *metrics.MetricFactory
}

// NewAgent creates a new agent instance
func NewAgent(config *config.Config, ansible ansible.Repository, orchestrator repository.ContainerAppRepository) (*Agent, error) {
	c, err := client.NewClient(config, ansible, orchestrator)
	if err != nil {
		return nil, log.Errorf("New GRPC client failed", err)
	}

	start := time.Now()

	return &Agent{
		client:            c,
		config:            config,
		startTime:         start,
		metricsFactory:    metrics.NewMetricsFactory(start),
		systemInfoFactory: metrics.NewSystemInfoFactory(start),
	}, nil
}

// Register the agent with the server
func (a *Agent) Register() error {
	log.Printf("Getting system capabilities")
	capabilities := GetCapabilities().ToMap()

	log.Printf("Registering agent with server")
	resp, err := a.client.RegisterAgent(capabilities, a.config.Features, a.config.AgentID)
	if err != nil {
		return err
	}

	if resp.Base.ResponseCode != 1 { // RESPONSE_CODE_SUCCESS = 1
		return log.Errorf("registration failed: %s", resp.Base.Message)
	}

	log.Printf("Agent registered successfully")
	return nil
}

// collectMetrics collects system metrics for heartbeat
func (a *Agent) collectMetrics() map[string]string {
	return a.metricsFactory.Collect()
}

// collectSystemInfo collects system information for heartbeat
func (a *Agent) collectSystemInfo() map[string]string {
	return a.systemInfoFactory.Collect()
}

// StartHeartbeat starts the heartbeat stream
func (a *Agent) StartHeartbeat() error {
	log.Printf("Collecting system metrics for heartbeat")

	log.Printf("Getting system capabilities for heartbeat")
	capabilities := GetCapabilities().ToMap()

	log.Printf("Starting heartbeat stream with server")
	return a.client.StartAgentStream(
		a.config.AgentID,
		a.collectMetrics,
		capabilities,
		a.config.Features,
	)
}

// Close closes the agent's client connection
func (a *Agent) Close() {
	if a.client != nil {
		a.client.Close()
	}
}

// Run starts the agent's main loop
func (a *Agent) Run(ctx context.Context) error {
	// Silence unused warning if caller intentionally passes context for future use
	_ = ctx
	// Register the agent (client handles internal retry & backoff)
	log.Printf("Registering agent with server: %s", a.config.GetGRPCServerAddress())
	if err := a.Register(); err != nil {
		return log.Errorf("failed to register agent: %v", err)
	}
	log.Printf("Agent registered successfully")

	// Start heartbeat stream
	log.Printf("Starting heartbeat stream")
	if err := a.StartHeartbeat(); err != nil {
		return log.Errorf("failed to start heartbeat stream: %v", err)
	}
	log.Printf("Heartbeat stream started successfully")

	return nil
}
