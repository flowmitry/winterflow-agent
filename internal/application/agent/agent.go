package agent

import (
	"context"
	"time"
	"winterflow-agent/internal/application"
	"winterflow-agent/internal/application/command"
	"winterflow-agent/internal/application/query"
	"winterflow-agent/pkg/log"

	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/infra/winterflow/grpc/client"
	"winterflow-agent/pkg/cqrs"
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
func NewAgent(ctx context.Context, config *config.Config) (*Agent, error) {
	appRepository := application.NewAppRepository(config)
	registryRepository := application.NewRegistryRepository()
	networkRepository := application.NewNetworkRepository()

	// Create command bus and register handlers
	commandBus := cqrs.NewCommandBus(ctx)
	if err := command.RegisterCommandHandlers(commandBus, config, appRepository, registryRepository, networkRepository); err != nil {
		log.Fatalf("Failed to register command handlers: %v", err)
	}

	// Create query bus and register handlers
	queryBus := cqrs.NewQueryBus(ctx)
	if err := query.RegisterQueryHandlers(queryBus, config, appRepository, registryRepository, networkRepository); err != nil {
		log.Fatalf("Failed to register query handlers: %v", err)
	}

	c, err := client.NewClient(ctx, config, commandBus, queryBus)
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

// registerAgent the agent with the server
func (a *Agent) registerAgent(ctx context.Context, capabilities map[string]string) error {
	log.Debug("Registering agent with server")
	resp, err := a.client.RegisterAgent(ctx, capabilities, a.config.Features, a.config.AgentID)
	if err != nil {
		return err
	}

	if resp.Base.ResponseCode != 1 { // RESPONSE_CODE_SUCCESS = 1
		return log.Errorf("registration failed: %s", resp.Base.Message)
	}

	log.Info("Agent registered successfully")
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

func (a *Agent) startAgentStream(ctx context.Context, capabilities map[string]string) error {
	return a.client.StartAgentStream(
		ctx,
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
	capabilities := GetCapabilities().ToMap()
	log.Info("Registering agent with server", "server_address", a.config.GetGRPCServerAddress())
	if err := a.registerAgent(ctx, capabilities); err != nil {
		return log.Errorf("failed to register agent: %v", err)
	}
	log.Info("Agent registered successfully")

	// Start heartbeat stream
	log.Info("Starting agent's stream")
	if err := a.startAgentStream(ctx, capabilities); err != nil {
		return log.Errorf("failed to start heartbeat stream: %v", err)
	}
	log.Info("Heartbeat stream started successfully")

	return nil
}
