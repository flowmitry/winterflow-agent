package agent

import (
	"flag"
)

// Config holds the application configuration
type Config struct {
	ShowVersion   bool
	ShowHelp      bool
	ServerAddress string
	ServerID      string
	ServerToken   string
}

// NewConfig creates and parses command line flags
func NewConfig() *Config {
	config := &Config{}

	flag.BoolVar(&config.ShowVersion, "version", false, "Show version information")
	flag.BoolVar(&config.ShowHelp, "help", false, "Show help")
	flag.StringVar(&config.ServerAddress, "server", "localhost:8081", "gRPC server address")
	flag.StringVar(&config.ServerID, "id", "452cf5be-0f05-463c-bfc8-5dc9e39e1745", "Agent server ID")
	flag.StringVar(&config.ServerToken, "token", "server-token", "Server token for registration")

	flag.Parse()
	return config
}
