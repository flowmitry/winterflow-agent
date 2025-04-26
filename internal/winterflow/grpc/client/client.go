package client

import (
	"context"
	"log"
	"time"

	"winterflow-agent/internal/winterflow/grpc/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client represents a gRPC client for agent communication
type Client struct {
	conn   *grpc.ClientConn
	client pb.AgentServiceClient
	ctx    context.Context
	cancel context.CancelFunc
}

// NewClient creates a new gRPC client
func NewClient(serverAddress string) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		cancel()
		return nil, err
	}

	client := pb.NewAgentServiceClient(conn)

	return &Client{
		conn:   conn,
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	c.cancel()
	return c.conn.Close()
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

	return c.client.RegisterAgentV1(c.ctx, req)
}

// UnregisterAgent unregisters the agent from the server
func (c *Client) UnregisterAgent(serverID, accessToken string) (*pb.UnregisterAgentResponseV1, error) {
	req := &pb.UnregisterAgentRequestV1{
		ServerId:    serverID,
		AccessToken: accessToken,
	}

	return c.client.UnregisterAgentV1(c.ctx, req)
}

// StartHeartbeatStream starts a bidirectional stream for heartbeat communication
func (c *Client) StartHeartbeatStream(serverID, accessToken string, metrics map[string]string) error {
	stream, err := c.client.AgentStreamV1(c.ctx)
	if err != nil {
		return err
	}

	// Send initial heartbeat
	heartbeat := &pb.AgentHeartbeatV1{
		ServerId:    serverID,
		AccessToken: accessToken,
		Metrics:     metrics,
	}

	if err := stream.Send(heartbeat); err != nil {
		return err
	}

	// Start goroutine to receive responses
	go func() {
		for {
			response, err := stream.Recv()
			if err != nil {
				log.Printf("Error receiving heartbeat response: %v", err)
				return
			}

			if !response.Success {
				log.Printf("Heartbeat failed: %s", response.Message)
			}
		}
	}()

	// Start goroutine to send periodic heartbeats
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				heartbeat := &pb.AgentHeartbeatV1{
					ServerId:    serverID,
					AccessToken: accessToken,
					Metrics:     metrics,
				}

				if err := stream.Send(heartbeat); err != nil {
					log.Printf("Error sending heartbeat: %v", err)
					return
				}
			case <-c.ctx.Done():
				return
			}
		}
	}()

	return nil
}
