package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"winterflow-agent/internal/winterflow/grpc/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	// Default reconnection parameters
	defaultReconnectInterval        = 5 * time.Second
	defaultMaximumReconnectInterval = 320 * time.Second
	defaultConnectionTimeout        = 30 * time.Second
)

// Client represents a gRPC client for agent communication
type Client struct {
	conn   *grpc.ClientConn
	client pb.AgentServiceClient
	ctx    context.Context
	cancel context.CancelFunc

	// Reconnection parameters
	serverAddress            string
	reconnectInterval        time.Duration
	maximumReconnectInterval time.Duration
	reconnectAttempts        int
	connectionTimeout        time.Duration
}

// NewClient creates a new gRPC client
func NewClient(serverAddress string) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	log.Printf("Creating new gRPC client for %s", serverAddress)

	// Create gRPC client with insecure credentials and connection timeout
	clientConn, err := grpc.NewClient(
		serverAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(defaultConnectionTimeout),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create gRPC client: %v", err)
	}

	// Create temporary client for initial connection
	tempClient := &Client{
		conn:                     clientConn,
		ctx:                      ctx,
		cancel:                   cancel,
		serverAddress:            serverAddress,
		reconnectInterval:        defaultReconnectInterval,
		maximumReconnectInterval: defaultMaximumReconnectInterval,
		reconnectAttempts:        0,
		connectionTimeout:        defaultConnectionTimeout,
	}

	// Wait for the connection to be ready with endless retries
	if err := tempClient.waitForConnectionReady(); err != nil {
		clientConn.Close()
		cancel()
		return nil, fmt.Errorf("failed to establish initial connection: %v", err)
	}

	client := pb.NewAgentServiceClient(clientConn)

	return &Client{
		conn:   clientConn,
		client: client,
		ctx:    ctx,
		cancel: cancel,

		// Initialize reconnection parameters
		serverAddress:            serverAddress,
		reconnectInterval:        defaultReconnectInterval,
		maximumReconnectInterval: defaultMaximumReconnectInterval,
		reconnectAttempts:        0,
		connectionTimeout:        defaultConnectionTimeout,
	}, nil
}

// SetReconnectParameters sets custom reconnection parameters
func (c *Client) SetReconnectParameters(initialInterval, maxInterval time.Duration) {
	c.reconnectInterval = initialInterval
	c.maximumReconnectInterval = maxInterval
}

// SetConnectionTimeout sets the connection timeout
func (c *Client) SetConnectionTimeout(timeout time.Duration) {
	c.connectionTimeout = timeout
}

// Close closes the client connection
func (c *Client) Close() error {
	c.cancel()
	return c.conn.Close()
}

// getNextReconnectInterval calculates the next reconnection interval using exponential backoff
func (c *Client) getNextReconnectInterval() time.Duration {
	// Calculate exponential backoff: interval * 2^attempts
	interval := c.reconnectInterval * time.Duration(1<<uint(c.reconnectAttempts))

	// Cap the interval at the maximum
	if interval > c.maximumReconnectInterval {
		interval = c.maximumReconnectInterval
	}

	// Increment attempt counter for next time
	c.reconnectAttempts++

	return interval
}

// waitForConnectionReady waits for the connection to be ready with endless retries
func (c *Client) waitForConnectionReady() error {
	log.Printf("Waiting for connection to be ready")

	// Start connection attempt
	c.conn.Connect()

	for {
		state := c.conn.GetState()
		log.Printf("Current connection state: %v", state)

		if state == connectivity.Ready {
			log.Printf("Connection is ready")
			return nil
		}
		if state == connectivity.Shutdown {
			return fmt.Errorf("connection is shutdown")
		}
		if state == connectivity.TransientFailure {
			log.Printf("Connection attempt failed, retrying with exponential backoff")
			// Wait before retrying using exponential backoff
			nextInterval := c.getNextReconnectInterval()
			log.Printf("Waiting %v before next connection attempt (attempt %d)",
				nextInterval, c.reconnectAttempts)
			time.Sleep(nextInterval)
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

// reconnect attempts to reconnect to the server
func (c *Client) reconnect() error {
	log.Printf("Attempting to reconnect to %s (attempt %d)",
		c.serverAddress, c.reconnectAttempts+1)

	// Close existing connection if it exists
	if c.conn != nil {
		log.Printf("Closing existing connection")
		c.conn.Close()
	}

	// Create new connection using NewClient
	clientConn, err := grpc.NewClient(
		c.serverAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(c.connectionTimeout),
	)
	if err != nil {
		c.reconnectAttempts++
		return fmt.Errorf("failed to create new connection: %v", err)
	}

	// Update client state
	c.conn = clientConn
	c.client = pb.NewAgentServiceClient(clientConn)

	// Wait for the connection to be ready with endless retries
	if err := c.waitForConnectionReady(); err != nil {
		clientConn.Close()
		c.reconnectAttempts++
		return fmt.Errorf("failed to establish connection: %v", err)
	}

	c.reconnectAttempts = 0
	log.Printf("Successfully reconnected to %s", c.serverAddress)
	return nil
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

// RegisterAgent registers the agent with the server
func (c *Client) RegisterAgent(version string, capabilities map[string]string, features map[string]bool, serverID, serverToken string) (*pb.RegisterAgentResponseV1, error) {
	req := &pb.RegisterAgentRequestV1{
		Version:      version,
		Capabilities: capabilities,
		Features:     features,
		ServerId:     serverID,
		ServerToken:  serverToken,
	}

	for {
		// Ensure connection is ready before making the request
		if err := c.waitForReady(); err != nil {
			log.Printf("Connection not ready before registration: %v", err)
			if err := c.reconnect(); err != nil {
				log.Printf("Failed to reconnect, will retry: %v", err)
				time.Sleep(c.getNextReconnectInterval())
				continue
			}
		}

		resp, err := c.client.RegisterAgentV1(c.ctx, req)
		if err != nil {
			if status.Code(err) == codes.Unavailable {
				log.Printf("Connection unavailable during registration, attempting to reconnect")
				if err := c.reconnect(); err != nil {
					log.Printf("Failed to reconnect, will retry: %v", err)
					time.Sleep(c.getNextReconnectInterval())
					continue
				}
				continue
			}
			// For other errors, log and retry
			log.Printf("Error during registration, will retry: %v", err)
			time.Sleep(c.getNextReconnectInterval())
			continue
		}

		return resp, nil
	}
}

// UnregisterAgent unregisters the agent from the server
func (c *Client) UnregisterAgent(serverID, accessToken string) (*pb.UnregisterAgentResponseV1, error) {
	req := &pb.UnregisterAgentRequestV1{
		ServerId:    serverID,
		AccessToken: accessToken,
	}

	// Ensure connection is ready before making the request
	if err := c.waitForReady(); err != nil {
		log.Printf("Connection not ready before unregistration: %v", err)
		return nil, fmt.Errorf("connection not ready: %v", err)
	}

	resp, err := c.client.UnregisterAgentV1(c.ctx, req)
	if err != nil {
		if status.Code(err) == codes.Unavailable {
			log.Printf("Connection unavailable during unregistration: %v", err)
			return nil, fmt.Errorf("connection unavailable: %v", err)
		}
		return nil, err
	}

	return resp, nil
}

// StartHeartbeatStream starts a bidirectional stream for heartbeat communication
func (c *Client) StartHeartbeatStream(serverID, accessToken string, metrics map[string]string, version string, capabilities map[string]string, features map[string]bool, serverToken string) error {
	// Start goroutine to maintain the heartbeat stream
	go func() {
		for {
			// Ensure connection is ready before starting the stream
			if err := c.waitForReady(); err != nil {
				log.Printf("Connection not ready before starting heartbeat stream: %v", err)
				if err := c.reconnect(); err != nil {
					log.Printf("Failed to reconnect, will retry: %v", err)
					time.Sleep(c.getNextReconnectInterval())
					continue
				}
				// After reconnection, wait for connection to be ready
				if err := c.waitForReady(); err != nil {
					log.Printf("Connection still not ready after reconnection: %v", err)
					time.Sleep(c.getNextReconnectInterval())
					continue
				}
			}

			stream, err := c.client.AgentStreamV1(c.ctx)
			if err != nil {
				log.Printf("Failed to create heartbeat stream: %v", err)
				if err := c.reconnect(); err != nil {
					log.Printf("Failed to reconnect, will retry: %v", err)
					time.Sleep(c.getNextReconnectInterval())
					continue
				}
				// After reconnection, wait for connection to be ready
				if err := c.waitForReady(); err != nil {
					log.Printf("Connection still not ready after reconnection: %v", err)
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
				Metrics:     metrics,
			}

			if err := stream.Send(heartbeat); err != nil {
				log.Printf("Failed to send initial heartbeat: %v", err)
				if err := c.reconnect(); err != nil {
					log.Printf("Failed to reconnect, will retry: %v", err)
					time.Sleep(c.getNextReconnectInterval())
					continue
				}
				// After reconnection, wait for connection to be ready
				if err := c.waitForReady(); err != nil {
					log.Printf("Connection still not ready after reconnection: %v", err)
					time.Sleep(c.getNextReconnectInterval())
					continue
				}
				continue
			}

			log.Printf("Initial heartbeat sent successfully")

			// Channel to signal when re-registration is needed
			reregisterCh := make(chan struct{}, 1)
			// Channel to signal fatal errors
			fatalErrorCh := make(chan error, 1)
			// Channel to signal stream recreation
			recreateStreamCh := make(chan struct{}, 1)
			// Channel to signal stream cleanup
			cleanupCh := make(chan struct{}, 1)
			// Channel to signal stream is ready
			streamReadyCh := make(chan struct{}, 1)

			// Start goroutine to receive responses
			go func() {
				for {
					response, err := stream.Recv()
					if err != nil {
						log.Printf("Error receiving heartbeat response: %v", err)

						// Check if it's a connection error or EOF
						if status.Code(err) == codes.Unavailable || err == io.EOF {
							log.Printf("Connection unavailable or stream closed in heartbeat stream: %v", err)
							// Trigger stream recreation
							select {
							case recreateStreamCh <- struct{}{}:
							default:
							}
							return // Exit to trigger stream recreation
						}

						// For other errors, log and continue
						log.Printf("Non-connection error in heartbeat stream: %v", err)
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

					case pb.ResponseCode_RESPONSE_CODE_SERVER_NOT_FOUND:
						log.Printf("Server not found, this is a fatal error")
						select {
						case fatalErrorCh <- fmt.Errorf("server not found"):
						default:
						}
						return

					case pb.ResponseCode_RESPONSE_CODE_AGENT_ALREADY_CONNECTED:
						log.Printf("Agent already connected, this is a fatal error")
						select {
						case fatalErrorCh <- fmt.Errorf("agent already connected"):
						default:
						}
						return

					default:
						if !response.Success {
							log.Printf("Heartbeat failed: %s", response.Message)
							// Check if the error message indicates token expiration
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

			// Start goroutine to send periodic heartbeats
			go func() {
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()

				// Wait for stream to be ready
				<-streamReadyCh

				for {
					select {
					case <-ticker.C:
						heartbeat := &pb.AgentHeartbeatV1{
							ServerId:    serverID,
							AccessToken: accessToken,
							Metrics:     metrics,
						}

						log.Printf("Sending periodic heartbeat with metrics: %v", metrics)

						if err := stream.Send(heartbeat); err != nil {
							log.Printf("Error sending heartbeat: %v", err)

							// Check if it's a connection error or EOF
							if status.Code(err) == codes.Unavailable || err == io.EOF {
								log.Printf("Connection unavailable or stream closed in heartbeat stream: %v", err)
								// Trigger stream recreation
								select {
								case recreateStreamCh <- struct{}{}:
								default:
								}
								return // Exit to trigger stream recreation
							}

							// For other errors, log and continue
							log.Printf("Non-connection error in heartbeat stream: %v", err)
						} else {
							log.Printf("Periodic heartbeat sent successfully")
						}
					case <-cleanupCh:
						log.Printf("Stopping heartbeat sender due to stream cleanup")
						return
					case <-c.ctx.Done():
						return
					}
				}
			}()

			// Signal that stream is ready for heartbeats
			select {
			case streamReadyCh <- struct{}{}:
			default:
			}

			// Wait for re-registration signal, stream recreation, fatal error, or context cancellation
			select {
			case <-reregisterCh:
				log.Printf("Re-registering agent due to token expiration or agent not found")
				// Signal cleanup of current stream
				select {
				case cleanupCh <- struct{}{}:
				default:
				}
				// Close the current stream
				stream.CloseSend()
				// Attempt to re-register
				resp, err := c.RegisterAgent(version, capabilities, features, serverID, serverToken)
				if err != nil {
					log.Printf("Failed to re-register agent: %v", err)
					time.Sleep(c.getNextReconnectInterval())
					continue
				}
				// Update access token from re-registration response
				accessToken = resp.AccessToken
				log.Printf("Successfully re-registered agent with new token")
				// Continue outer loop to create new stream
				continue

			case <-recreateStreamCh:
				log.Printf("Recreating heartbeat stream due to connection issues")
				// Signal cleanup of current stream
				select {
				case cleanupCh <- struct{}{}:
				default:
				}
				// Close the current stream
				stream.CloseSend()
				// Continue outer loop to create new stream
				continue

			case err := <-fatalErrorCh:
				log.Printf("Fatal error in heartbeat stream: %v", err)
				// Signal cleanup of current stream
				select {
				case cleanupCh <- struct{}{}:
				default:
				}
				// Close the current stream
				stream.CloseSend()
				return

			case <-c.ctx.Done():
				// Signal cleanup of current stream
				select {
				case cleanupCh <- struct{}{}:
				default:
				}
				// Close the current stream
				stream.CloseSend()
				return
			}
		}
	}()

	return nil
}
