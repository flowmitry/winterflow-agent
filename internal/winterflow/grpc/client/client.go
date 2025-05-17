package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
	"winterflow-agent/internal/winterflow/handlers"
	log "winterflow-agent/pkg/log"

	"winterflow-agent/internal/winterflow/grpc/pb"
	"winterflow-agent/pkg/certs"
	"winterflow-agent/pkg/cqrs"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"
	"winterflow-agent/pkg/backoff"
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

	// Certificate paths
	certPath string
	keyPath  string
	useTLS   bool
}

// setupConnection creates a new gRPC connection and client
func (c *Client) setupConnection() error {
	var opts []grpc.DialOption

	// Add timeout option
	opts = append(opts, grpc.WithTimeout(c.connectionTimeout))

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

	log.Printf("Setting up secure gRPC connection with TLS credentials")
	creds, err := certs.LoadTLSCredentials(c.certPath, c.keyPath)
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
func NewClient(serverAddress string, certPath, keyPath string) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	log.Printf("Creating new gRPC client for %s", serverAddress)

	// Create command bus and register handlers
	commandBus := cqrs.NewCommandBus()
	if err := handlers.RegisterCommandHandlers(commandBus); err != nil {
		cancel()
		return nil, log.Errorf("failed to register command handlers: %v", err)
	}

	// Create query bus and register handlers
	queryBus := cqrs.NewQueryBus()
	if err := handlers.RegisterQueryHandlers(queryBus); err != nil {
		cancel()
		return nil, log.Errorf("failed to register query handlers: %v", err)
	}

	// Always use TLS and fail if certificates don't exist
	if certPath == "" || keyPath == "" {
		return nil, log.Errorf("TLS is required but certificate paths are not configured")
	}

	if !certs.CertificateExists(certPath) {
		return nil, log.Errorf("TLS is required but certificate does not exist at path: %s", certPath)
	}

	if !certs.CertificateExists(keyPath) {
		return nil, log.Errorf("TLS is required but key does not exist at path: %s", keyPath)
	}

	log.Printf("TLS enabled with certificate: %s", certPath)

	client := &Client{
		ctx:               ctx,
		cancel:            cancel,
		serverAddress:     serverAddress,
		connectionTimeout: DefaultConnectionTimeout,
		streamCleanup:     make(chan struct{}),
		isRegistered:      true,
		regMutex:          sync.RWMutex{},
		backoffStrategy:   backoff.New(DefaultReconnectInterval, DefaultMaximumReconnectInterval),
		commandBus:        commandBus,
		queryBus:          queryBus,
		certPath:          certPath,
		keyPath:           keyPath,
		useTLS:            true, // Always use TLS
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

	attemptCount := 0
	for {
		attemptCount++
		log.Debug("Connection attempt #%d", attemptCount)

		// Check for context cancellation first
		select {
		case <-c.ctx.Done():
			return log.Errorf("connection cancelled: %v", c.ctx.Err())
		default:
			// Continue with normal operation
		}

		state := c.conn.GetState()
		log.Printf("Current connection state: %v (attempt #%d)", state, attemptCount)

		// Log detailed information based on connection state
		switch state {
		case connectivity.Ready:
			log.Printf("Connection is ready after %d attempts (total time: %v)", attemptCount, time.Since(startTime))
			// Reset the backoff sequence because the connection has been
			// successfully re-established. This prevents the next transient
			// failure from starting with an unnecessarily long delay and keeps
			// the reconnection behaviour predictable and responsive.
			c.backoffStrategy.Reset()
			return nil

		case connectivity.Shutdown:
			return log.Errorf("connection is shutdown after %d attempts (total time: %v)", attemptCount, time.Since(startTime))

		case connectivity.TransientFailure:
			log.Printf("Connection attempt #%d failed with TransientFailure, retrying with exponential backoff", attemptCount)

			// Wait before retrying using exponential backoff
			nextInterval := c.getNextReconnectInterval()
			log.Printf("Waiting %v before next connection attempt (attempt #%d)", nextInterval, attemptCount+1)

			// Use a timer so we can interrupt the wait
			timer := time.NewTimer(nextInterval)
			select {
			case <-timer.C:
				log.Debug("Backoff timer expired, proceeding with next connection attempt")
				// Timer expired, continue with next attempt
			case <-c.ctx.Done():
				// Context cancelled, abort reconnection
				timer.Stop()
				return log.Errorf("connection cancelled during reconnection: %v", c.ctx.Err())
			}

			// Trigger another connection attempt
			log.Debug("Initiating new connection attempt #%d after backoff", attemptCount+1)
			c.conn.Connect()
			continue

		case connectivity.Connecting:
			log.Debug("Connection is in CONNECTING state (attempt #%d, elapsed time: %v)", attemptCount, time.Since(startTime))

		case connectivity.Idle:
			log.Debug("Connection is in IDLE state (attempt #%d, elapsed time: %v)", attemptCount, time.Since(startTime))

		default:
			log.Debug("Connection is in unknown state: %v (attempt #%d, elapsed time: %v)", state, attemptCount, time.Since(startTime))
		}

		// Use a timer so we can interrupt the wait
		log.Debug("Waiting 100ms before checking connection state again")
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-c.ctx.Done():
			timer.Stop()
			return log.Errorf("connection cancelled: %v", c.ctx.Err())
		case <-timer.C:
			continue
		}
	}
}

// waitForReady waits for the connection to be ready
func (c *Client) waitForReady() error {
	log.Debug("Checking if connection to %s is ready", c.serverAddress)
	startTime := time.Now()

	state := c.conn.GetState()
	log.Printf("Current connection state to %s: %v (checked in %v)", c.serverAddress, state, time.Since(startTime))

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
func (c *Client) RegisterAgent(capabilities map[string]string, features map[string]bool, serverID string) (*pb.RegisterAgentResponseV1, error) {
	log.Printf("Starting agent registration process")

	// Create a unique message ID
	messageID := GenerateUUID()

	req := &pb.RegisterAgentRequestV1{
		MessageId:    messageID,
		Timestamp:    TimestampNow(),
		ServerId:     serverID,
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
			log.Printf("Connection not ready before registration: %v", err)
			if err := c.reconnect(); err != nil {
				log.Printf("Failed to reconnect, will retry: %v", err)

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

		log.Printf("Sending RegisterAgentV1 request")
		resp, err := c.client.RegisterAgentV1(c.ctx, req)
		if err != nil {
			grpcCode := status.Code(err)
			switch grpcCode {
			case codes.FailedPrecondition:
				return nil, ErrUnrecoverable
			case codes.AlreadyExists:
				return nil, ErrUnrecoverableAgentAlreadyConnected
			case codes.Unavailable:
				log.Printf("Connection unavailable during registration, attempting to reconnect")
				if err := c.reconnect(); err != nil {
					log.Printf("Failed to reconnect, will retry: %v", err)
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
				log.Printf("Error during registration, will retry: %v", err)
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
				log.Printf("Registration failed with response code %v, retrying", resp.Base.ResponseCode)
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
		log.Printf("Registration successful, setting registered state")
		c.SetRegistered(true)

		return resp, nil
	}
}

// StartAgentStream starts a bidirectional stream
func (c *Client) StartAgentStream(serverID string, metricsProvider func() map[string]string, capabilities map[string]string, features map[string]bool) error {
	log.Printf("Starting Agent stream with server ID: %s", serverID)
	log.Printf("Current registration state: %v", c.IsRegistered())

	// Start goroutine to maintain the heartbeat stream
	go func() {
		log.Printf("Agent stream goroutine started")
	outerLoop:
		for {
			// Check if we should stop
			select {
			case <-c.streamCleanup:
				log.Printf("Agent stream cleanup requested")
				return
			default:
			}

			// Check if agent is still registered
			if !c.IsRegistered() {
				log.Printf("Agent is not registered, stopping Agent stream")
				return
			}

			// Ensure connection is ready before starting the stream
			if err := c.waitForReady(); err != nil {
				log.Printf("Connection not ready before starting Agent stream: %v", err)
				if err := c.reconnect(); err != nil {
					log.Printf("Failed to reconnect, will retry: %v", err)

					// Use a timer so we can interrupt the wait
					timer := time.NewTimer(c.getNextReconnectInterval())
					select {
					case <-timer.C:
						// Timer expired, continue with next attempt
					case <-c.ctx.Done():
						// Context cancelled, abort reconnection
						timer.Stop()
						log.Printf("Stream cancelled during reconnection: %v", c.ctx.Err())
						return
					}
					continue
				}
				continue
			}

			log.Printf("Creating Agent stream")
			stream, err := c.client.AgentStream(c.ctx)
			if err != nil {
				log.Printf("Failed to create Agent stream: %v", err)
				if err := c.reconnect(); err != nil {
					log.Printf("Failed to reconnect, will retry: %v", err)

					// Use a timer so we can interrupt the wait
					timer := time.NewTimer(c.getNextReconnectInterval())
					select {
					case <-timer.C:
						// Timer expired, continue with next attempt
					case <-c.ctx.Done():
						// Context cancelled, abort reconnection
						timer.Stop()
						log.Printf("Stream cancelled during reconnection: %v", c.ctx.Err())
						return
					}
					continue
				}
				continue
			}

			log.Printf("Agent stream established successfully")

			// Send initial heartbeat
			baseMsg := &pb.BaseMessage{
				MessageId: GenerateUUID(),
				Timestamp: TimestampNow(),
				ServerId:  serverID,
			}

			heartbeat := &pb.AgentHeartbeatV1{
				Base:    baseMsg,
				Metrics: metricsProvider(),
			}

			agentMsg := &pb.AgentMessage{
				Message: &pb.AgentMessage_HeartbeatV1{
					HeartbeatV1: heartbeat,
				},
			}

			if err := stream.Send(agentMsg); err != nil {
				log.Printf("Failed to send initial heartbeat: %v", err)
				if err := c.reconnect(); err != nil {
					log.Printf("Failed to reconnect, will retry: %v", err)

					// Use a timer so we can interrupt the wait
					timer := time.NewTimer(c.getNextReconnectInterval())
					select {
					case <-timer.C:
						// Timer expired, continue with next attempt
					case <-c.ctx.Done():
						// Context cancelled, abort reconnection
						timer.Stop()
						log.Printf("Stream cancelled during reconnection: %v", c.ctx.Err())
						return
					}
					continue
				}
				continue
			}

			log.Printf("Initial heartbeat sent successfully")

			// Create channels for stream management
			streamDone := make(chan struct{})
			reregisterCh := make(chan struct{})
			fatalErrorCh := make(chan error)
			appRequestCh := make(chan *pb.GetAppRequestV1)
			createAppRequestCh := make(chan *pb.CreateAppRequestV1)

			// Start goroutine to receive responses
			go func() {
				defer close(streamDone)
				for {
					serverCmd, err := stream.Recv()
					if err != nil {
						if status.Code(err) == codes.Unavailable || err == io.EOF {
							log.Printf("Connection unavailable or stream closed: %v", err)
							return
						}
						log.Printf("Error receiving server command: %v", err)
						continue
					}

					// Handle different command types
					switch cmd := serverCmd.Command.(type) {
					case *pb.ServerCommand_HeartbeatResponseV1:
						response := cmd.HeartbeatResponseV1.Base

						// Handle response codes
						switch response.ResponseCode {
						case pb.ResponseCode_RESPONSE_CODE_AGENT_NOT_FOUND:
							log.Printf("Agent not found, triggering re-registration")
							select {
							case reregisterCh <- struct{}{}:
							default:
							}
							return

						case pb.ResponseCode_RESPONSE_CODE_SERVER_NOT_FOUND,
							pb.ResponseCode_RESPONSE_CODE_AGENT_ALREADY_CONNECTED:
							log.Printf("Received response code %v, triggering re-registration", response.ResponseCode)
							select {
							case reregisterCh <- struct{}{}:
							default:
							}
							return

						case pb.ResponseCode_RESPONSE_CODE_SUCCESS:
							log.Printf("Heartbeat response received: %s", response.Message)

						default:
							log.Error("Heartbeat failed with code %v: %s", response.ResponseCode, response.Message)
						}

					case *pb.ServerCommand_GetAppRequestV1:
						log.Printf("Received app request: %s", cmd.GetAppRequestV1.Base.MessageId)
						// Forward the request to be handled by the main loop
						select {
						case appRequestCh <- cmd.GetAppRequestV1:
						default:
							log.Printf("Warning: App request channel full, dropping request")
						}

					case *pb.ServerCommand_CreateAppRequestV1:
						log.Printf("Received create app request: %s", cmd.CreateAppRequestV1.Base.MessageId)
						// Forward the request to be handled by the main loop
						select {
						case createAppRequestCh <- cmd.CreateAppRequestV1:
						default:
							log.Printf("Warning: Create app request channel full, dropping request")
						}

					default:
						// Log details about the unknown command type
						log.Printf("Received unknown command type: %T", cmd)
					}
				}
			}()

			// Start periodic heartbeat sender
			ticker := time.NewTicker(HeartbeatInterval)

			for {
				select {
				case <-ticker.C:
					if !c.IsRegistered() {
						log.Printf("Agent is not registered, stopping heartbeat sender")
						return
					}

					baseMsg := &pb.BaseMessage{
						MessageId: GenerateUUID(),
						Timestamp: TimestampNow(),
						ServerId:  serverID,
					}

					heartbeat := &pb.AgentHeartbeatV1{
						Base:    baseMsg,
						Metrics: metricsProvider(),
					}

					agentMsg := &pb.AgentMessage{
						Message: &pb.AgentMessage_HeartbeatV1{
							HeartbeatV1: heartbeat,
						},
					}

					if err := stream.Send(agentMsg); err != nil {
						log.Printf("Error sending heartbeat: %v", err)
						if status.Code(err) == codes.Unavailable || err == io.EOF {
							return
						}
						continue
					}
					log.Printf("Periodic heartbeat sent successfully")

				case appRequest := <-appRequestCh:
					agentMsg, err := HandleGetAppQuery(c.queryBus, appRequest, serverID)
					if err != nil {
						log.Error("Error retrieving app response: %v", err)
						continue
					}

					if err := stream.Send(agentMsg); err != nil {
						log.Warn("Error sending app response: %v", err)
						continue
					}
					log.Info("App response sent successfully")

				case createAppRequest := <-createAppRequestCh:
					agentMsg, err := HandleCreateAppRequest(c.commandBus, createAppRequest, serverID)
					if err != nil {
						log.Error("Error creating app response: %v", err)
						continue
					}

					if err := stream.Send(agentMsg); err != nil {
						log.Warn("Error sending create app response: %v", err)
						continue
					}
					log.Info("Create app response sent successfully")

				case <-streamDone:
					log.Printf("Stream receiver stopped, recreating stream")
					ticker.Stop()
					continue outerLoop

				case <-reregisterCh:
					log.Printf("Re-registering agent due to agent not found")
					stream.CloseSend()
					_, err := c.RegisterAgent(capabilities, features, serverID)
					if err != nil {
						log.Printf("Failed to re-register agent: %v", err)
						ticker.Stop()

						// Use a timer so we can interrupt the wait
						timer := time.NewTimer(c.getNextReconnectInterval())
						select {
						case <-timer.C:
							// Timer expired, continue with next attempt
						case <-c.ctx.Done():
							// Context cancelled, abort reconnection
							timer.Stop()
							log.Printf("Stream cancelled during re-registration: %v", c.ctx.Err())
							return
						}
						continue outerLoop
					}
					log.Printf("Successfully re-registered agent")
					ticker.Stop()
					continue outerLoop

				case err := <-fatalErrorCh:
					log.Printf("Fatal error in heartbeat stream: %v", err)
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
	log.Printf("Attempting to reconnect to %s", c.serverAddress)
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

	log.Printf("Successfully reconnected to %s (total time: %v)", c.serverAddress, time.Since(startTime))
	return nil
}
