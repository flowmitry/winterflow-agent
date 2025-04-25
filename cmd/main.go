package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"winterflow-agent/pkg/agent"
	"winterflow-agent/pkg/grpc/client"
)

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	serverAddress := flag.String("server", "localhost:8081", "gRPC server address")
	serverID := flag.String("id", "452cf5be-0f05-463c-bfc8-5dc9e39e1745", "Agent server ID")
	serverToken := flag.String("token", "server-token", "Server token for registration")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("WinterFlow Agent version: %s (#%d)\n", agent.GetVersion(), agent.GetNumericVersion())
		os.Exit(0)
	}

	// Create a new client
	c, err := client.NewClient(*serverAddress)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	// Register the agent
	capabilities := map[string]string{
		"os":      "darwin",
		"version": agent.GetVersion(),
	}

	resp, err := c.RegisterAgent(agent.GetVersion(), capabilities, *serverID, *serverToken)
	if err != nil {
		log.Fatalf("Failed to register agent: %v", err)
	}

	if !resp.Success {
		log.Fatalf("Registration failed: %s", resp.Message)
	}

	log.Printf("Agent registered successfully. Access token: %s", resp.AccessToken)

	// Start heartbeat stream
	metrics := map[string]string{
		"cpu_usage": "0.5",
		"memory":    "512MB",
	}

	err = c.StartHeartbeatStream(*serverID, resp.AccessToken, metrics)
	if err != nil {
		log.Fatalf("Failed to start heartbeat stream: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Unregister the agent
	unregisterResp, err := c.UnregisterAgent(*serverID, resp.AccessToken)
	if err != nil {
		log.Printf("Failed to unregister agent: %v", err)
	} else if !unregisterResp.Success {
		log.Printf("Unregistration failed: %s", unregisterResp.Message)
	} else {
		log.Println("Agent unregistered successfully")
	}
}
