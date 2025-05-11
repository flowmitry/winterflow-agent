package client

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	log "winterflow-agent/pkg/log"

	"winterflow-agent/internal/winterflow/grpc/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
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

	// Access token storage
	accessToken string

	// Stream cleanup
	streamCleanup chan struct{}

	// Registration state
	isRegistered bool
	regMutex     sync.RWMutex
}

// setupConnection creates a new gRPC connection and client
func (c *Client) setupConnection() error {
	clientConn, err := grpc.NewClient(
		c.serverAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(c.connectionTimeout),
	)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %v", err)
	}

	c.conn = clientConn
	c.client = pb.NewAgentServiceClient(clientConn)
	return nil
}

// NewClient creates a new gRPC client
func NewClient(serverAddress string) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	log.Printf("Creating new gRPC client for %s", serverAddress)

	client := &Client{
		ctx:               ctx,
		cancel:            cancel,
		serverAddress:     serverAddress,
		connectionTimeout: DefaultConnectionTimeout,
		streamCleanup:     make(chan struct{}),
		isRegistered:      true,
		regMutex:          sync.RWMutex{},
		backoffStrategy:   backoff.New(DefaultReconnectInterval, DefaultMaximumReconnectInterval),
	}

	if err := client.setupConnection(); err != nil {
		cancel()
		return nil, err
	}

	// Wait for the connection to be ready with endless retries
	if err := client.waitForConnectionReady(); err != nil {
		client.conn.Close()
		cancel()
		return nil, fmt.Errorf("failed to establish initial connection: %v", err)
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

// SetAccessToken sets the access token for the client
func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
}

// GetAccessToken returns the current access token
func (c *Client) GetAccessToken() string {
	return c.accessToken
}

// Close closes the client connection
func (c *Client) Close() error {
	c.cancel()
	return c.conn.Close()
}

// getNextReconnectInterval calculates the next reconnection interval using exponential backoff
func (c *Client) getNextReconnectInterval() time.Duration {
	return c.backoffStrategy.Next()
}

// waitForConnectionReady waits for the connection to be ready with endless retries
func (c *Client) waitForConnectionReady() error {
	log.Printf("Waiting for connection to be ready")

	// Start connection attempt
	c.conn.Connect()

	for {
		// Check for context cancellation first
		select {
		case <-c.ctx.Done():
			return fmt.Errorf("connection cancelled: %v", c.ctx.Err())
		default:
			// Continue with normal operation
		}

		state := c.conn.GetState()
		log.Printf("Current connection state: %v", state)

		if state == connectivity.Ready {
			log.Printf("Connection is ready")
			// Reset the backoff sequence because the connection has been
			// successfully re-established. This prevents the next transient
			// failure from starting with an unnecessarily long delay and keeps
			// the reconnection behaviour predictable and responsive.
			c.backoffStrategy.Reset()
			return nil
		}
		if state == connectivity.Shutdown {
			return fmt.Errorf("connection is shutdown")
		}
		if state == connectivity.TransientFailure {
			log.Printf("Connection attempt failed, retrying with exponential backoff")
			// Wait before retrying using exponential backoff
			nextInterval := c.getNextReconnectInterval()
			log.Printf("Waiting %v before next connection attempt", nextInterval)

			// Use a timer so we can interrupt the wait
			timer := time.NewTimer(nextInterval)
			select {
			case <-timer.C:
				// Timer expired, continue with next attempt
			case <-c.ctx.Done():
				// Context cancelled, abort reconnection
				timer.Stop()
				return fmt.Errorf("connection cancelled during reconnection: %v", c.ctx.Err())
			}

			// Trigger another connection attempt
			c.conn.Connect()
			continue
		}

		select {
		case <-c.ctx.Done():
			return fmt.Errorf("connection cancelled: %v", c.ctx.Err())
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
}

// waitForReady waits for the connection to be ready
func (c *Client) waitForReady() error {
	state := c.conn.GetState()
	log.Printf("Checking connection state: %v", state)

	if state == connectivity.Ready {
		return nil
	}
	if state == connectivity.Shutdown {
		return fmt.Errorf("connection is shutdown")
	}
	if state == connectivity.TransientFailure {
		return fmt.Errorf("connection is in transient failure: server may be unavailable at %s",
			c.conn.Target())
	}
	return fmt.Errorf("connection is not ready: %v", state)
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
func (c *Client) RegisterAgent(capabilities map[string]string, features map[string]bool, serverID, serverToken string) (*pb.RegisterAgentResponseV1, error) {
	log.Printf("Starting agent registration process")

	// Create a unique message ID
	messageID := GenerateUUID()

	req := &pb.RegisterAgentRequestV1{
		MessageId:    messageID,
		Timestamp:    TimestampNow(),
		ServerId:     serverID,
		ServerToken:  serverToken,
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
		c.accessToken = resp.AccessToken

		return resp, nil
	}
}

// StartAgentStream starts a bidirectional stream
func (c *Client) StartAgentStream(serverID, accessToken string, metricsProvider func() map[string]string, capabilities map[string]string, features map[string]bool, serverToken string) error {
	log.Printf("Starting heartbeat stream with server ID: %s", serverID)
	log.Printf("Current registration state: %v", c.IsRegistered())

	// Start goroutine to maintain the heartbeat stream
	go func() {
		log.Printf("Heartbeat stream goroutine started")
	outerLoop:
		for {
			// Check if we should stop
			select {
			case <-c.streamCleanup:
				log.Printf("Heartbeat stream cleanup requested")
				return
			default:
			}

			// Check if agent is still registered
			if !c.IsRegistered() {
				log.Printf("Agent is not registered, stopping heartbeat stream")
				return
			}

			// Ensure connection is ready before starting the stream
			if err := c.waitForReady(); err != nil {
				log.Printf("Connection not ready before starting heartbeat stream: %v", err)
				if err := c.reconnect(); err != nil {
					log.Printf("Failed to reconnect, will retry: %v", err)
					time.Sleep(c.getNextReconnectInterval())
					continue
				}
				continue
			}

			log.Printf("Creating heartbeat stream")
			stream, err := c.client.AgentStream(c.ctx)
			if err != nil {
				log.Printf("Failed to create heartbeat stream: %v", err)
				if err := c.reconnect(); err != nil {
					log.Printf("Failed to reconnect, will retry: %v", err)
					time.Sleep(c.getNextReconnectInterval())
					continue
				}
				continue
			}

			log.Printf("Heartbeat stream established successfully")

			// Send initial heartbeat
			baseMsg := &pb.BaseMessage{
				MessageId:   GenerateUUID(),
				Timestamp:   TimestampNow(),
				ServerId:    serverID,
				AccessToken: accessToken,
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
					time.Sleep(c.getNextReconnectInterval())
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
							log.Printf("Heartbeat failed with code %v: %s", response.ResponseCode, response.Message)
							if strings.Contains(response.Message, "token expired") ||
								strings.Contains(response.Message, "Invalid token") {
								log.Printf("Token expired or invalid, triggering re-registration")
								select {
								case reregisterCh <- struct{}{}:
								default:
								}
								return
							}
						}

					case *pb.ServerCommand_GetAppRequestV1:
						log.Printf("Received app request: %s", cmd.GetAppRequestV1.Base.MessageId)
						// Forward the request to be handled by the main loop
						select {
						case appRequestCh <- cmd.GetAppRequestV1:
						default:
							log.Printf("Warning: App request channel full, dropping request")
						}

					default:
						log.Printf("Received unknown command type")
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
						MessageId:   GenerateUUID(),
						Timestamp:   TimestampNow(),
						ServerId:    serverID,
						AccessToken: accessToken,
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
					log.Printf("Processing app request for app ID: %s", appRequest.AppId)

					// Here you would implement the logic to retrieve the app
					// For now, we'll just send a dummy response
					baseResp := &pb.BaseResponse{
						MessageId:    GenerateUUID(),
						Timestamp:    TimestampNow(),
						ResponseCode: pb.ResponseCode_RESPONSE_CODE_SUCCESS,
						Message:      "App retrieved successfully",
						ServerId:     serverID,
					}

					appResp := &pb.GetAppResponseV1{
						Base: baseResp,
						App: &pb.AppResponseV1{
							AppId: appRequest.AppId,
							// Other fields would be populated here
						},
					}

					agentMsg := &pb.AgentMessage{
						Message: &pb.AgentMessage_GetAppResponseV1{
							GetAppResponseV1: appResp,
						},
					}

					if err := stream.Send(agentMsg); err != nil {
						log.Printf("Error sending app response: %v", err)
						continue
					}
					log.Printf("App response sent successfully")

				case <-streamDone:
					log.Printf("Stream receiver stopped, recreating stream")
					ticker.Stop()
					continue outerLoop

				case <-reregisterCh:
					log.Printf("Re-registering agent due to token expiration or agent not found")
					stream.CloseSend()
					resp, err := c.RegisterAgent(capabilities, features, serverID, serverToken)
					if err != nil {
						log.Printf("Failed to re-register agent: %v", err)
						ticker.Stop()
						time.Sleep(c.getNextReconnectInterval())
						continue outerLoop
					}
					accessToken = resp.AccessToken
					log.Printf("Successfully re-registered agent with new token")
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

	// Close existing connection if it exists
	if c.conn != nil {
		log.Printf("Closing existing connection")
		c.conn.Close()
	}

	if err := c.setupConnection(); err != nil {
		return err
	}

	// Wait for the connection to be ready with endless retries
	if err := c.waitForConnectionReady(); err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to establish connection: %v", err)
	}

	// Reset the backoff sequence after a successful reconnection.
	c.backoffStrategy.Reset()

	log.Printf("Successfully reconnected to %s", c.serverAddress)
	return nil
}
