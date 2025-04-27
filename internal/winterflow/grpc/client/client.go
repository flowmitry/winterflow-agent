package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"winterflow-agent/internal/winterflow/grpc/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"winterflow-agent/pkg/backoff"
)

const (
	// Default reconnection parameters
	defaultReconnectInterval        = 5 * time.Second
	defaultMaximumReconnectInterval = 320 * time.Second
	defaultConnectionTimeout        = 30 * time.Second
	heartbeatInterval               = 5 * time.Second // unified heartbeat cadence
)

// ErrUnrecoverable is returned by RegisterAgent when the server indicates that
// the agent must not retry the registration (e.g. wrong server-token pairing
// or duplicate agent).
var ErrUnrecoverableServerNotFound = errors.New("unrecoverable registration error: server not found")
var ErrUnrecoverableAgentAlreadyConnected = errors.New("unrecoverable registration error: agent already connected")
var ErrUnrecoverableAgentNotFound = errors.New("unrecoverable registration error: agent not found")

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
		connectionTimeout: defaultConnectionTimeout,
		streamCleanup:     make(chan struct{}),
		isRegistered:      true,
		regMutex:          sync.RWMutex{},
		backoffStrategy:   backoff.New(defaultReconnectInterval, defaultMaximumReconnectInterval),
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
func (c *Client) RegisterAgent(version string, capabilities map[string]string, features map[string]bool, serverID, serverToken string) (*pb.RegisterAgentResponseV1, error) {
	log.Printf("Starting agent registration process")

	req := &pb.RegisterAgentRequestV1{
		Version:      version,
		Capabilities: capabilities,
		Features:     features,
		ServerId:     serverID,
		ServerToken:  serverToken,
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
			// If the server explicitly tells that the agent already exists or the
			// server/agent is not found we treat it as unrecoverable and abort
			// further retries so that the caller can exit.
			if grpcCode == codes.AlreadyExists {
				return nil, ErrUnrecoverableAgentAlreadyConnected
			}
			if grpcCode == codes.NotFound {
				return nil, ErrUnrecoverableAgentNotFound
			}

			if grpcCode == codes.Unavailable {
				log.Printf("Connection unavailable during registration, attempting to reconnect")
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
				continue
			}
			// For other errors, log and retry
			log.Printf("Error during registration, will retry: %v", err)

			// Use a timer so we can interrupt the wait
			timer := time.NewTimer(c.getNextReconnectInterval())
			select {
			case <-timer.C:
				// Timer expired, continue with next attempt
			case <-c.ctx.Done():
				// Context cancelled, abort reconnection
				timer.Stop()
				return nil, fmt.Errorf("registration cancelled during retry: %v", c.ctx.Err())
			}

			continue
		}

		// Evaluate response codes that indicate unrecoverable registration errors.
		switch resp.ResponseCode {
		case pb.ResponseCode_RESPONSE_CODE_SERVER_NOT_FOUND:
			return nil, ErrUnrecoverableServerNotFound
		case pb.ResponseCode_RESPONSE_CODE_AGENT_ALREADY_CONNECTED:
			return nil, ErrUnrecoverableAgentAlreadyConnected
		case pb.ResponseCode_RESPONSE_CODE_AGENT_NOT_FOUND:
			return nil, ErrUnrecoverableAgentNotFound
		}

		// Success path.
		log.Printf("Registration successful, setting registered state")
		c.SetRegistered(true)
		c.accessToken = resp.AccessToken

		return resp, nil
	}
}

// StartHeartbeatStream starts a bidirectional stream for heartbeat communication
func (c *Client) StartHeartbeatStream(serverID, accessToken string, metricsProvider func() map[string]string, version string, capabilities map[string]string, features map[string]bool, serverToken string) error {
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
			stream, err := c.client.AgentStreamV1(c.ctx)
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
			heartbeat := &pb.AgentHeartbeatV1{
				ServerId:    serverID,
				AccessToken: accessToken,
				Metrics:     metricsProvider(),
			}

			if err := stream.Send(heartbeat); err != nil {
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

			// Start goroutine to receive responses
			go func() {
				defer close(streamDone)
				for {
					response, err := stream.Recv()
					if err != nil {
						if status.Code(err) == codes.Unavailable || err == io.EOF {
							log.Printf("Connection unavailable or stream closed: %v", err)
							return
						}
						log.Printf("Error receiving heartbeat response: %v", err)
						continue
					}

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

					default:
						if !response.Success {
							log.Printf("Heartbeat failed: %s", response.Message)
							if strings.Contains(response.Message, "token expired") ||
								strings.Contains(response.Message, "Invalid token") {
								log.Printf("Token expired or invalid, triggering re-registration")
								select {
								case reregisterCh <- struct{}{}:
								default:
								}
								return
							}
						} else {
							log.Printf("Heartbeat response received: %s", response.Message)
						}
					}
				}
			}()

			// Start periodic heartbeat sender
			ticker := time.NewTicker(heartbeatInterval)

			for {
				select {
				case <-ticker.C:
					if !c.IsRegistered() {
						log.Printf("Agent is not registered, stopping heartbeat sender")
						return
					}

					heartbeat := &pb.AgentHeartbeatV1{
						ServerId:    serverID,
						AccessToken: accessToken,
						Metrics:     metricsProvider(),
					}

					if err := stream.Send(heartbeat); err != nil {
						log.Printf("Error sending heartbeat: %v", err)
						if status.Code(err) == codes.Unavailable || err == io.EOF {
							return
						}
						continue
					}
					log.Printf("Periodic heartbeat sent successfully")

				case <-streamDone:
					log.Printf("Stream receiver stopped, recreating stream")
					ticker.Stop()
					continue outerLoop

				case <-reregisterCh:
					log.Printf("Re-registering agent due to token expiration or agent not found")
					stream.CloseSend()
					resp, err := c.RegisterAgent(version, capabilities, features, serverID, serverToken)
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
