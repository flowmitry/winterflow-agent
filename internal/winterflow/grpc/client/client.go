package client

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/ansible"
	"winterflow-agent/internal/winterflow/handlers"
	log "winterflow-agent/pkg/log"

	"winterflow-agent/internal/winterflow/grpc/pb"
	"winterflow-agent/pkg/certs"
	"winterflow-agent/pkg/cqrs"

	"winterflow-agent/pkg/backoff"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"
)

const (
	// queueChannelSize defines the buffer size for a channel used to queue tasks or data within the system.
	queueChannelSize = 1
)

// Client represents a gRPC client for agent communication
type Client struct {
	conn   *grpc.ClientConn
	client pb.AgentServiceClient
	ctx    context.Context
	cancel context.CancelFunc

	// Reconnection and timeouts
	serverAddress     string
	connectionTimeout time.Duration

	// Exponential back-off helper for reconnection attempts to keep the code
	// DRY and easier to maintain.
	backoffStrategy *backoff.Backoff

	// Stream cleanup
	streamCleanup chan struct{}

	// Registration state
	isRegistered bool
	regMutex     sync.RWMutex

	// Command and Query buses for CQRS
	commandBus cqrs.CommandBus
	queryBus   cqrs.QueryBus

	config *config.Config

	// Certificate paths
	caCertPath string
	certPath   string
	keyPath    string

	// Reconnect mutex
	reconnectMu sync.Mutex
}

// setupConnection creates a new gRPC connection and client
func (c *Client) setupConnection() error {
	var opts []grpc.DialOption

	// Add timeout option
	opts = append(opts, grpc.WithTimeout(c.connectionTimeout))
	// Ensure dial blocks so the timeout is respected
	opts = append(opts, grpc.WithBlock())

	// Always use TLS and fail if certificates don't exist
	if c.certPath == "" || c.keyPath == "" {
		return log.Errorf("TLS is required but certificate paths are not configured")
	}

	if !certs.CertificateExists(c.certPath) {
		return log.Errorf("TLS is required but certificate does not exist at path: %s", c.certPath)
	}

	if !certs.CertificateExists(c.keyPath) {
		return log.Errorf("TLS is required but key does not exist at path: %s", c.keyPath)
	}

	log.Info("Setting up secure gRPC connection with TLS credentials")
	host, _, err := net.SplitHostPort(c.serverAddress)
	if err != nil {
		host = c.serverAddress
	}
	creds, err := certs.LoadTLSCredentials(c.caCertPath, c.certPath, c.keyPath, host)
	if err != nil {
		return log.Errorf("Failed to load TLS credentials: %v", err)
	}
	opts = append(opts, grpc.WithTransportCredentials(creds))

	clientConn, err := grpc.NewClient(
		c.serverAddress,
		opts...,
	)
	if err != nil {
		return log.Errorf("failed to create gRPC client: %v", err)
	}

	c.conn = clientConn
	c.client = pb.NewAgentServiceClient(clientConn)
	return nil
}

// NewClient creates a new gRPC client
func NewClient(config *config.Config, ansible ansible.Repository) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())
	serverAddress := config.GetGRPCServerAddress()
	caCertPath := config.GetCACertificatePath()
	certPath := config.GetCertificatePath()
	keyPath := config.GetPrivateKeyPath()

	log.Info("Creating new gRPC client", "serverAddress", serverAddress)

	// Create command bus and register handlers
	commandBus := cqrs.NewCommandBus()
	if err := handlers.RegisterCommandHandlers(commandBus, config, ansible); err != nil {
		cancel()
		return nil, log.Errorf("failed to register command handlers: %v", err)
	}

	// Create query bus and register handlers
	queryBus := cqrs.NewQueryBus()
	if err := handlers.RegisterQueryHandlers(queryBus, config, ansible); err != nil {
		cancel()
		return nil, log.Errorf("failed to register query handlers: %v", err)
	}

	// Always use TLS and fail if certificates don't exist
	if caCertPath == "" || certPath == "" || keyPath == "" {
		return nil, log.Errorf("TLS is required but certificate paths are not configured")
	}

	if !certs.CertificateExists(caCertPath) {
		return nil, log.Errorf("TLS is required but CA certificate does not exist at path: %s", caCertPath)
	}

	if !certs.CertificateExists(certPath) {
		return nil, log.Errorf("TLS is required but certificate does not exist at path: %s", certPath)
	}

	if !certs.CertificateExists(keyPath) {
		return nil, log.Errorf("TLS is required but key does not exist at path: %s", keyPath)
	}

	log.Info("TLS enabled", "certificate", certPath)

	client := &Client{
		ctx:               ctx,
		cancel:            cancel,
		serverAddress:     serverAddress,
		connectionTimeout: DefaultConnectionTimeout,
		streamCleanup:     make(chan struct{}),
		isRegistered:      false,
		regMutex:          sync.RWMutex{},
		backoffStrategy:   backoff.New(DefaultReconnectInterval, DefaultMaximumReconnectInterval),
		commandBus:        commandBus,
		queryBus:          queryBus,
		caCertPath:        caCertPath,
		certPath:          certPath,
		keyPath:           keyPath,
		config:            config,
	}

	if err := client.setupConnection(); err != nil {
		cancel()
		return nil, err
	}

	// Wait for the connection to be ready with endless retries
	if err := client.waitForConnectionReady(); err != nil {
		client.conn.Close()
		cancel()
		return nil, log.Errorf("failed to establish initial connection: %v", err)
	}

	return client, nil
}

// SetReconnectParameters sets custom reconnection parameters
func (c *Client) SetReconnectParameters(initialInterval, maxInterval time.Duration) {
	c.backoffStrategy = backoff.New(initialInterval, maxInterval)
}

// SetConnectionTimeout sets the connection timeout
func (c *Client) SetConnectionTimeout(timeout time.Duration) {
	c.connectionTimeout = timeout
}

// Close closes the client connection and gracefully shuts down the command and query buses
func (c *Client) Close() error {
	// Initiate graceful shutdown of the command and query buses
	c.commandBus.Shutdown()
	c.queryBus.Shutdown()

	// Cancel the context to stop any ongoing operations
	c.cancel()

	// Wait for all active commands and queries to complete
	c.commandBus.WaitForCompletion()
	c.queryBus.WaitForCompletion()

	// Close the gRPC connection
	return c.conn.Close()
}

// getNextReconnectInterval calculates the next reconnection interval using exponential backoff
func (c *Client) getNextReconnectInterval() time.Duration {
	return c.backoffStrategy.Next()
}

// waitForConnectionReady waits for the connection to be ready with endless retries
func (c *Client) waitForConnectionReady() error {
	log.Debug("Waiting for connection to be ready to server: %s", c.serverAddress)

	// Start connection attempt
	log.Debug("Initiating connection attempt to %s", c.serverAddress)
	startTime := time.Now()
	c.conn.Connect()
	log.Debug("Connection attempt initiated, took %v", time.Since(startTime))

	// First connection attempt already issued above.
	attemptCount := 1
	log.Debug("Connection attempt", "attempt", attemptCount)

	for {
		// Check for context cancellation first
		select {
		case <-c.ctx.Done():
			return log.Errorf("connection cancelled: %v", c.ctx.Err())
		default:
			// Continue with normal operation
		}

		// Capture current connection state
		state := c.conn.GetState()
		log.Debug("Current connection state", "state", state, "attempt", attemptCount)

		switch state {
		case connectivity.Ready:
			log.Info("Connection is ready", "attempts", attemptCount, "totalTime", time.Since(startTime))
			return nil

		case connectivity.Shutdown:
			return log.Errorf("connection is shutdown after %d attempts (total time: %v)", attemptCount, time.Since(startTime))

		case connectivity.TransientFailure:
			// The previous dial attempt resulted in a transient failure, apply backoff before retrying.
			log.Warn("Connection attempt failed with TransientFailure", "attempt", attemptCount, "action", "retrying with exponential backoff")

			nextInterval := c.getNextReconnectInterval()
			log.Info("Waiting before next connection attempt", "waitTime", nextInterval, "nextAttempt", attemptCount+1)

			timer := time.NewTimer(nextInterval)
			select {
			case <-timer.C:
				// Proceed with the next attempt
			case <-c.ctx.Done():
				timer.Stop()
				return log.Errorf("connection cancelled during reconnection: %v", c.ctx.Err())
			}

			// Start a new connection attempt after backoff.
			attemptCount++
			log.Debug("Initiating new connection attempt", "attempt", attemptCount)
			c.conn.Connect()
			continue

		case connectivity.Connecting, connectivity.Idle:
			// For CONNECTING and IDLE states we simply wait for the state to change.
			// gRPC will transition either to READY, TRANSIENT_FAILURE or SHUTDOWN.
			log.Debug("Waiting for state change", "currentState", state)

			if !c.conn.WaitForStateChange(c.ctx, state) {
				// Context was cancelled while waiting.
				return log.Errorf("connection cancelled while waiting for state change: %v", c.ctx.Err())
			}
			// State changed, loop and evaluate again.
			continue

		default:
			// Unknown state, wait for a state change.
			log.Debug("Encountered unknown connection state, waiting for change", "state", state)
			if !c.conn.WaitForStateChange(c.ctx, state) {
				return log.Errorf("connection cancelled while waiting for state change: %v", c.ctx.Err())
			}
			continue
		}
	}
}

// waitForReady waits for the connection to be ready
func (c *Client) waitForReady() error {
	log.Debug("Checking if connection to %s is ready", c.serverAddress)
	startTime := time.Now()

	state := c.conn.GetState()
	log.Info("Current connection state", "serverAddress", c.serverAddress, "state", state, "checkTime", time.Since(startTime))

	switch state {
	case connectivity.Ready:
		log.Debug("Connection to %s is ready", c.serverAddress)
		return nil

	case connectivity.Shutdown:
		return log.Errorf("Connection to %s is shutdown", c.serverAddress)

	case connectivity.TransientFailure:
		target := c.conn.Target()
		return log.Errorf("Connection to %s is in transient failure: server may be unavailable at %s",
			c.serverAddress, target)

	case connectivity.Connecting:
		return log.Errorf("Connection to %s is still connecting", c.serverAddress)

	case connectivity.Idle:
		return log.Errorf("Connection to %s is idle and not ready", c.serverAddress)

	default:
		return log.Errorf("Connection to %s is not ready: %v (unknown state)", c.serverAddress, state)
	}
}

// IsRegistered returns whether the agent is currently registered
func (c *Client) IsRegistered() bool {
	c.regMutex.RLock()
	defer c.regMutex.RUnlock()
	return c.isRegistered
}

// SetRegistered sets the registration state
func (c *Client) SetRegistered(registered bool) {
	c.regMutex.Lock()
	defer c.regMutex.Unlock()
	c.isRegistered = registered
}

// RegisterAgent registers the agent with the server
func (c *Client) RegisterAgent(capabilities map[string]string, features map[string]bool, agentID string) (*pb.RegisterAgentResponseV1, error) {
	log.Info("Starting agent registration process")

	// Create a unique message ID
	messageID := GenerateUUID()

	req := &pb.RegisterAgentRequestV1{
		Base: &pb.BaseMessage{
			MessageId: messageID,
			Timestamp: TimestampNow(),
			AgentId:   agentID,
		},
		Capabilities: capabilities,
		Features:     features,
	}

	for {
		// Check for context cancellation first
		select {
		case <-c.ctx.Done():
			return nil, fmt.Errorf("registration cancelled: %v", c.ctx.Err())
		default:
			// Continue with normal operation
		}

		// Ensure connection is ready before making the request
		if err := c.waitForReady(); err != nil {
			log.Warn("Connection not ready before registration", "error", err)
			if err := c.reconnect(); err != nil {
				log.Warn("Failed to reconnect, will retry", "error", err)

				// Use a timer so we can interrupt the wait
				timer := time.NewTimer(c.getNextReconnectInterval())
				select {
				case <-timer.C:
					// Timer expired, continue with next attempt
				case <-c.ctx.Done():
					// Context cancelled, abort reconnection
					timer.Stop()
					return nil, fmt.Errorf("registration cancelled during reconnection: %v", c.ctx.Err())
				}

				continue
			}
		}

		log.Info("Sending RegisterAgentV1 request")
		resp, err := c.client.RegisterAgentV1(c.ctx, req)
		if err != nil {
			grpcCode := status.Code(err)
			switch grpcCode {
			case codes.FailedPrecondition:
				return nil, ErrUnrecoverable
			case codes.AlreadyExists:
				return nil, ErrUnrecoverableAgentAlreadyConnected
			case codes.Unavailable:
				log.Warn("Connection unavailable during registration", "action", "attempting to reconnect")
				if err := c.reconnect(); err != nil {
					log.Warn("Failed to reconnect, will retry", "error", err)
					timer := time.NewTimer(c.getNextReconnectInterval())
					select {
					case <-timer.C:
					case <-c.ctx.Done():
						timer.Stop()
						return nil, fmt.Errorf("registration cancelled during reconnection: %v", c.ctx.Err())
					}
				}
				continue
			default:
				log.Warn("Error during registration", "error", err, "action", "will retry")
				timer := time.NewTimer(c.getNextReconnectInterval())
				select {
				case <-timer.C:
				case <-c.ctx.Done():
					timer.Stop()
					return nil, fmt.Errorf("registration cancelled during retry: %v", c.ctx.Err())
				}
				continue
			}
		}

		// Handle application-level response codes
		if resp.Base.ResponseCode != pb.ResponseCode_RESPONSE_CODE_SUCCESS {
			switch resp.Base.ResponseCode {
			case pb.ResponseCode_RESPONSE_CODE_AGENT_ALREADY_CONNECTED:
				return nil, ErrUnrecoverableAgentAlreadyConnected
			default:
				log.Warn("Registration failed", "responseCode", resp.Base.ResponseCode, "action", "retrying")
				timer := time.NewTimer(c.getNextReconnectInterval())
				select {
				case <-timer.C:
				case <-c.ctx.Done():
					timer.Stop()
					return nil, fmt.Errorf("registration cancelled during retry: %v", c.ctx.Err())
				}
				continue
			}
		}

		// Success path.
		log.Info("Registration successful", "action", "setting registered state")
		c.SetRegistered(true)

		return resp, nil
	}
}

// StartAgentStream starts a bidirectional stream
func (c *Client) StartAgentStream(agentID string, metricsProvider func() map[string]string, capabilities map[string]string, features map[string]bool) error {
	log.Info("Starting Agent stream", "agentID", agentID)
	log.Debug("Current registration state", "registered", c.IsRegistered())

	// Start goroutine to maintain the heartbeat stream
	go func() {
		log.Debug("Agent stream goroutine started")
	outerLoop:
		for {
			// Check if we should stop
			select {
			case <-c.streamCleanup:
				log.Info("Agent stream cleanup requested")
				return
			default:
			}

			// Check if agent is still registered
			if !c.IsRegistered() {
				log.Warn("Agent is not registered, stopping Agent stream")
				return
			}

			// Ensure connection is ready before starting the stream
			if err := c.waitForReady(); err != nil {
				log.Warn("Connection not ready before starting Agent stream", "error", err)
				if err := c.reconnect(); err != nil {
					log.Error("Failed to reconnect, will retry", "error", err)

					// Use a timer so we can interrupt the wait
					timer := time.NewTimer(c.getNextReconnectInterval())
					select {
					case <-timer.C:
						// Timer expired, continue with next attempt
					case <-c.ctx.Done():
						// Context cancelled, abort reconnection
						timer.Stop()
						log.Warn("Stream cancelled during reconnection", "error", c.ctx.Err())
						return
					}
					continue
				}
				continue
			}

			log.Debug("Creating Agent stream")
			stream, err := c.client.AgentStream(c.ctx)
			if err != nil {
				log.Error("Failed to create Agent stream", "error", err)
				if err := c.reconnect(); err != nil {
					log.Warn("Failed to reconnect, will retry", "error", err)

					// Use a timer so we can interrupt the wait
					timer := time.NewTimer(c.getNextReconnectInterval())
					select {
					case <-timer.C:
						// Timer expired, continue with next attempt
					case <-c.ctx.Done():
						// Context cancelled, abort reconnection
						timer.Stop()
						log.Warn("Stream cancelled during reconnection", "error", c.ctx.Err())
						return
					}
					continue
				}
				continue
			}

			log.Info("Agent stream established successfully")

			// Send initial heartbeat
			baseMsg := &pb.BaseMessage{
				MessageId: GenerateUUID(),
				Timestamp: TimestampNow(),
				AgentId:   agentID,
			}

			var metrics map[string]string
			if c.config.IsFeatureEnabled(config.FeatureSendMetricsDisabled) {
				metrics = make(map[string]string)
			} else {
				metrics = metricsProvider()
			}

			heartbeat := &pb.AgentHeartbeatV1{
				Base:    baseMsg,
				Metrics: metrics,
			}

			agentMsg := &pb.AgentMessage{
				Message: &pb.AgentMessage_HeartbeatV1{
					HeartbeatV1: heartbeat,
				},
			}

			if err := stream.Send(agentMsg); err != nil {
				log.Error("Failed to send initial heartbeat", "error", err)
				if status.Code(err) == codes.Unavailable || err == io.EOF {
					log.Warn("Connection unavailable or stream closed, recreating stream")
					continue outerLoop
				}
				if err := c.reconnect(); err != nil {
					log.Warn("Failed to reconnect, will retry", "error", err)

					// Use a timer so we can interrupt the wait
					timer := time.NewTimer(c.getNextReconnectInterval())
					select {
					case <-timer.C:
						// Timer expired, continue with next attempt
					case <-c.ctx.Done():
						// Context cancelled, abort reconnection
						timer.Stop()
						log.Warn("Stream cancelled during reconnection", "error", c.ctx.Err())
						return
					}
					continue
				}
				continue
			}

			log.Debug("Initial heartbeat sent successfully")

			// Create channels for stream management
			streamDone := make(chan struct{})
			reregisterCh := make(chan struct{})
			fatalErrorCh := make(chan error)
			appRequestCh := make(chan *pb.GetAppRequestV1, queueChannelSize)
			saveAppRequestCh := make(chan *pb.SaveAppRequestV1, queueChannelSize)
			deleteAppRequestCh := make(chan *pb.DeleteAppRequestV1, queueChannelSize)
			controlAppRequestCh := make(chan *pb.ControlAppRequestV1, queueChannelSize)
			getAppsStatusRequestCh := make(chan *pb.GetAppsStatusRequestV1, queueChannelSize)

			// Start goroutine to receive responses
			go func() {
				defer close(streamDone)
				for {
					serverCmd, err := stream.Recv()
					if err != nil {
						if status.Code(err) == codes.Unavailable || err == io.EOF {
							log.Error("Connection unavailable or stream closed", "error", err)
							log.Warn("Stream receiver stopping, will recreate stream")
							return
						}
						log.Error("Error receiving server command", "error", err)
						continue
					}

					// Handle different command types
					switch cmd := serverCmd.Command.(type) {
					case *pb.ServerCommand_HeartbeatResponseV1:
						response := cmd.HeartbeatResponseV1.Base

						// Handle response codes
						switch response.ResponseCode {
						case pb.ResponseCode_RESPONSE_CODE_AGENT_NOT_FOUND:
							log.Warn("Agent not found, triggering re-registration")
							select {
							case reregisterCh <- struct{}{}:
							default:
							}
							return

						case pb.ResponseCode_RESPONSE_CODE_AGENT_ALREADY_CONNECTED:
							log.Warn("Received response code, triggering re-registration", "code", response.ResponseCode)
							select {
							case reregisterCh <- struct{}{}:
							default:
							}
							return

						case pb.ResponseCode_RESPONSE_CODE_SUCCESS:
							log.Debug("Heartbeat response received", "message", response.Message)

						default:
							log.Error("Heartbeat failed", "code", response.ResponseCode, "message", response.Message)
						}

					case *pb.ServerCommand_UpdateAgentRequestV1:
						log.Info("Received update agent request", "messageId", cmd.UpdateAgentRequestV1.Base.MessageId)
						// Handle the update agent request directly since it will exit the process
						agentMsg, err := HandleUpdateAgentRequest(c.commandBus, cmd.UpdateAgentRequestV1, agentID)
						if err != nil {
							log.Error("Error handling update agent request", "error", err)
							continue
						}

						if err := stream.Send(agentMsg); err != nil {
							log.Error("Error sending update agent response", "error", err)
							if status.Code(err) == codes.Unavailable || err == io.EOF {
								log.Warn("Connection unavailable or stream closed, recreating stream")
								return
							}
							continue
						}
						log.Info("Update agent response sent successfully")

					case *pb.ServerCommand_GetAppRequestV1:
						log.Info("Received app request", "messageId", cmd.GetAppRequestV1.Base.MessageId)
						// Forward the request to be handled by the main loop
						select {
						case appRequestCh <- cmd.GetAppRequestV1:
						default:
							log.Warn("App request channel full, dropping request")
							// Create and send error response immediately
							baseResp := createBaseResponse(cmd.GetAppRequestV1.Base.MessageId, agentID, pb.ResponseCode_RESPONSE_CODE_TOO_MANY_REQUESTS, "Request dropped: channel full")
							getAppResp := &pb.GetAppResponseV1{
								Base:       &baseResp,
								App:        nil,
								AppVersion: cmd.GetAppRequestV1.AppVersion,
							}

							agentMsg := &pb.AgentMessage{
								Message: &pb.AgentMessage_GetAppResponseV1{
									GetAppResponseV1: getAppResp,
								},
							}

							if err := stream.Send(agentMsg); err != nil {
								log.Warn("Error sending dropped request response", "error", err)
							} else {
								log.Info("Dropped request response sent successfully")
							}
						}

					case *pb.ServerCommand_SaveAppRequestV1:
						log.Info("Received save app request", "messageId", cmd.SaveAppRequestV1.Base.MessageId)
						// Forward the request to be handled by the main loop
						select {
						case saveAppRequestCh <- cmd.SaveAppRequestV1:
						default:
							log.Warn("Save app request channel full, dropping request")
							// Create and send error response immediately
							baseResp := createBaseResponse(cmd.SaveAppRequestV1.Base.MessageId, agentID, pb.ResponseCode_RESPONSE_CODE_TOO_MANY_REQUESTS, "Request dropped: channel full")
							saveAppResp := &pb.SaveAppResponseV1{
								Base: &baseResp,
								App:  cmd.SaveAppRequestV1.App,
							}

							agentMsg := &pb.AgentMessage{
								Message: &pb.AgentMessage_SaveAppResponseV1{
									SaveAppResponseV1: saveAppResp,
								},
							}

							if err := stream.Send(agentMsg); err != nil {
								log.Warn("Error sending dropped request response", "error", err)
							} else {
								log.Info("Dropped request response sent successfully")
							}
						}

					case *pb.ServerCommand_DeleteAppRequestV1:
						log.Info("Received delete app request", "messageId", cmd.DeleteAppRequestV1.Base.MessageId)
						// Forward the request to be handled by the main loop
						select {
						case deleteAppRequestCh <- cmd.DeleteAppRequestV1:
						default:
							log.Warn("Delete app request channel full, dropping request")
							// Create and send error response immediately
							baseResp := createBaseResponse(cmd.DeleteAppRequestV1.Base.MessageId, agentID, pb.ResponseCode_RESPONSE_CODE_TOO_MANY_REQUESTS, "Request dropped: channel full")
							deleteAppResp := &pb.DeleteAppResponseV1{
								Base: &baseResp,
							}

							agentMsg := &pb.AgentMessage{
								Message: &pb.AgentMessage_DeleteAppResponseV1{
									DeleteAppResponseV1: deleteAppResp,
								},
							}

							if err := stream.Send(agentMsg); err != nil {
								log.Warn("Error sending dropped request response", "error", err)
							} else {
								log.Info("Dropped request response sent successfully")
							}
						}

					case *pb.ServerCommand_ControlAppRequestV1:
						log.Info("Received control app request", "messageId", cmd.ControlAppRequestV1.Base.MessageId)
						// Forward the request to be handled by the main loop
						select {
						case controlAppRequestCh <- cmd.ControlAppRequestV1:
						default:
							log.Warn("Control app request channel full, dropping request")
							// Create and send error response immediately
							baseResp := createBaseResponse(cmd.ControlAppRequestV1.Base.MessageId, agentID, pb.ResponseCode_RESPONSE_CODE_TOO_MANY_REQUESTS, "Request dropped: channel full")
							controlAppResp := &pb.ControlAppResponseV1{
								Base:       &baseResp,
								AppId:      cmd.ControlAppRequestV1.AppId,
								AppVersion: cmd.ControlAppRequestV1.AppVersion,
								StatusCode: pb.AppStatusCode_STATUS_CODE_PROBLEMATIC,
							}

							agentMsg := &pb.AgentMessage{
								Message: &pb.AgentMessage_ControlAppResponseV1{
									ControlAppResponseV1: controlAppResp,
								},
							}

							if err := stream.Send(agentMsg); err != nil {
								log.Warn("Error sending dropped request response", "error", err)
							} else {
								log.Info("Dropped request response sent successfully")
							}
						}

					case *pb.ServerCommand_GetAppsStatusRequestV1:
						log.Info("Received get apps status request", "messageId", cmd.GetAppsStatusRequestV1.Base.MessageId)
						// Forward the request to be handled by the main loop
						select {
						case getAppsStatusRequestCh <- cmd.GetAppsStatusRequestV1:
						default:
							log.Warn("Get apps status request channel full, dropping request")
							// Create and send error response immediately
							baseResp := createBaseResponse(cmd.GetAppsStatusRequestV1.Base.MessageId, agentID, pb.ResponseCode_RESPONSE_CODE_TOO_MANY_REQUESTS, "Request dropped: channel full")
							getAppsStatusResp := &pb.GetAppsStatusResponseV1{
								Base: &baseResp,
								Apps: nil,
							}

							agentMsg := &pb.AgentMessage{
								Message: &pb.AgentMessage_GetAppsStatusResponseV1{
									GetAppsStatusResponseV1: getAppsStatusResp,
								},
							}

							if err := stream.Send(agentMsg); err != nil {
								log.Warn("Error sending dropped request response", "error", err)
							} else {
								log.Info("Dropped request response sent successfully")
							}
						}

					default:
						// Log details about the unknown command type
						log.Warn("Received unknown command type", "type", fmt.Sprintf("%T", cmd))
					}
				}
			}()

			// Start periodic heartbeat sender
			ticker := time.NewTicker(HeartbeatInterval)

			for {
				select {
				case <-ticker.C:
					if !c.IsRegistered() {
						log.Warn("Agent is not registered, stopping heartbeat sender")
						return
					}

					baseMsg := &pb.BaseMessage{
						MessageId: GenerateUUID(),
						Timestamp: TimestampNow(),
						AgentId:   agentID,
					}

					var metrics map[string]string
					if sendMetricsDisabled, exists := features["send_metrics_disabled"]; exists && sendMetricsDisabled {
						metrics = make(map[string]string)
					} else {
						metrics = metricsProvider()
					}

					heartbeat := &pb.AgentHeartbeatV1{
						Base:    baseMsg,
						Metrics: metrics,
					}

					agentMsg := &pb.AgentMessage{
						Message: &pb.AgentMessage_HeartbeatV1{
							HeartbeatV1: heartbeat,
						},
					}

					if err := stream.Send(agentMsg); err != nil {
						log.Error("Error sending heartbeat", "error", err)
						if status.Code(err) == codes.Unavailable || err == io.EOF {
							log.Warn("Connection unavailable or stream closed, recreating stream")
							ticker.Stop()
							continue outerLoop
						}
						continue
					}
					log.Debug("Periodic heartbeat sent successfully")

				case appRequest := <-appRequestCh:
					agentMsg, err := HandleGetAppQuery(c.queryBus, appRequest, agentID)
					if err != nil {
						log.Error("Error retrieving app response", "error", err)
						continue
					}

					if err := stream.Send(agentMsg); err != nil {
						log.Error("Error sending app response", "error", err)
						if status.Code(err) == codes.Unavailable || err == io.EOF {
							log.Warn("Connection unavailable or stream closed, recreating stream")
							ticker.Stop()
							continue outerLoop
						}
						continue
					}
					log.Info("App response sent successfully")

				case saveAppRequest := <-saveAppRequestCh:
					agentMsg, err := HandleSaveAppRequest(c.commandBus, saveAppRequest, agentID)
					if err != nil {
						log.Error("Error saving app response", "error", err)
						continue
					}

					if err := stream.Send(agentMsg); err != nil {
						log.Error("Error sending save app response", "error", err)
						if status.Code(err) == codes.Unavailable || err == io.EOF {
							log.Warn("Connection unavailable or stream closed, recreating stream")
							ticker.Stop()
							continue outerLoop
						}
						continue
					}
					log.Info("Save app response sent successfully")

				case deleteAppRequest := <-deleteAppRequestCh:
					agentMsg, err := HandleDeleteAppRequest(c.commandBus, deleteAppRequest, agentID)
					if err != nil {
						log.Error("Error deleting app response", "error", err)
						continue
					}

					if err := stream.Send(agentMsg); err != nil {
						log.Error("Error sending delete app response", "error", err)
						if status.Code(err) == codes.Unavailable || err == io.EOF {
							log.Warn("Connection unavailable or stream closed, recreating stream")
							ticker.Stop()
							continue outerLoop
						}
						continue
					}
					log.Info("Delete app response sent successfully")

				case controlAppRequest := <-controlAppRequestCh:
					agentMsg, err := HandleControlAppRequest(c.commandBus, controlAppRequest, agentID)
					if err != nil {
						log.Error("Error controlling app response", "error", err)
						continue
					}

					if err := stream.Send(agentMsg); err != nil {
						log.Error("Error sending control app response", "error", err)
						if status.Code(err) == codes.Unavailable || err == io.EOF {
							log.Warn("Connection unavailable or stream closed, recreating stream")
							ticker.Stop()
							continue outerLoop
						}
						continue
					}
					log.Info("Control app response sent successfully")

				case getAppsStatusRequest := <-getAppsStatusRequestCh:
					agentMsg, err := HandleGetAppsStatusQuery(c.queryBus, getAppsStatusRequest, agentID)
					if err != nil {
						log.Error("Error retrieving apps statuses response", "error", err)
						continue
					}

					if err := stream.Send(agentMsg); err != nil {
						log.Error("Error sending get apps status response", "error", err)
						if status.Code(err) == codes.Unavailable || err == io.EOF {
							log.Warn("Connection unavailable or stream closed, recreating stream")
							ticker.Stop()
							continue outerLoop
						}
						continue
					}
					log.Info("Get apps status response sent successfully")

				case <-streamDone:
					log.Warn("Stream receiver stopped, recreating stream")
					ticker.Stop()
					continue outerLoop

				case <-reregisterCh:
					log.Warn("Re-registering agent due to agent not found")
					stream.CloseSend()
					_, err := c.RegisterAgent(capabilities, features, agentID)
					if err != nil {
						log.Error("Failed to re-register agent", "error", err)
						ticker.Stop()

						// Use a timer so we can interrupt the wait
						timer := time.NewTimer(c.getNextReconnectInterval())
						select {
						case <-timer.C:
							// Timer expired, continue with next attempt
						case <-c.ctx.Done():
							// Context cancelled, abort reconnection
							timer.Stop()
							log.Warn("Stream cancelled during re-registration", "error", c.ctx.Err())
							return
						}
						continue outerLoop
					}
					log.Info("Successfully re-registered agent")
					ticker.Stop()
					continue outerLoop

				case err := <-fatalErrorCh:
					log.Error("Fatal error in heartbeat stream", "error", err)
					stream.CloseSend()
					ticker.Stop()
					return

				case <-c.ctx.Done():
					stream.CloseSend()
					ticker.Stop()
					return
				}
			}
		}
	}()

	return nil
}

// reconnect attempts to reconnect to the server
func (c *Client) reconnect() error {
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	// If another goroutine already re-established the connection while we were waiting
	// for the lock, simply return without doing any work.
	if err := c.waitForReady(); err == nil {
		return nil
	}

	log.Info("Attempting to reconnect", "serverAddress", c.serverAddress)
	startTime := time.Now()

	// Close existing connection if it exists
	if c.conn != nil {
		log.Debug("Closing existing connection to %s", c.serverAddress)
		closeStartTime := time.Now()
		c.conn.Close()
		log.Debug("Existing connection closed in %v", time.Since(closeStartTime))
	} else {
		log.Debug("No existing connection to close")
	}

	// Always use TLS and fail if certificates don't exist
	log.Debug("Verifying TLS certificates before reconnection")
	if c.certPath == "" || c.keyPath == "" {
		return log.Errorf("TLS is required but certificate paths are not configured")
	}

	if !certs.CertificateExists(c.certPath) {
		return log.Errorf("TLS is required but certificate does not exist at path: %s", c.certPath)
	}

	if !certs.CertificateExists(c.keyPath) {
		return log.Errorf("TLS is required but key does not exist at path: %s", c.keyPath)
	}
	log.Debug("TLS certificates verified successfully")

	// Setup new connection
	log.Debug("Setting up new connection to %s", c.serverAddress)
	setupStartTime := time.Now()
	if err := c.setupConnection(); err != nil {
		return log.Errorf("Failed to setup connection: %v (took %v)", err, time.Since(setupStartTime))
	}
	log.Debug("Connection setup completed in %v", time.Since(setupStartTime))

	// Wait for the connection to be ready with endless retries
	log.Debug("Waiting for connection to be ready")
	waitStartTime := time.Now()
	if err := c.waitForConnectionReady(); err != nil {
		closeStartTime := time.Now()
		c.conn.Close()
		log.Debug("Connection closed in %v after failed wait", time.Since(closeStartTime))
		return fmt.Errorf("failed to establish connection: %v (waited for %v)", err, time.Since(waitStartTime))
	}
	log.Debug("Connection ready after waiting %v", time.Since(waitStartTime))

	// Reset the backoff sequence after a successful reconnection.
	c.backoffStrategy.Reset()
	log.Debug("Backoff strategy reset after successful reconnection")

	log.Info("Successfully reconnected", "serverAddress", c.serverAddress, "totalTime", time.Since(startTime))
	return nil
}
