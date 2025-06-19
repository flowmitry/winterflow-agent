package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"
	log "winterflow-agent/pkg/log"
)

// Config holds Docker client configuration
type Config struct {
	// SocketPath is the path to the Docker daemon socket
	SocketPath string
	// Timeout specifies the timeout for HTTP requests
	Timeout time.Duration
}

// DefaultConfig returns a default Docker client configuration
func DefaultConfig() *Config {
	socketPath := "/var/run/docker.sock"
	if runtime.GOOS == "windows" {
		socketPath = "\\\\.\\pipe\\docker_engine"
	}

	return &Config{
		SocketPath: socketPath,
		Timeout:    30 * time.Second,
	}
}

// Container represents a Docker container
type Container struct {
	ID      string            `json:"Id"`
	Names   []string          `json:"Names"`
	Image   string            `json:"Image"`
	ImageID string            `json:"ImageID"`
	Command string            `json:"Command"`
	Created int64             `json:"Created"`
	State   string            `json:"State"`
	Status  string            `json:"Status"`
	Ports   []Port            `json:"Ports"`
	Labels  map[string]string `json:"Labels"`
	Mounts  []Mount           `json:"Mounts"`
}

// Port represents a container port mapping
type Port struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}

// Mount represents a container mount
type Mount struct {
	Type        string `json:"Type"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Mode        string `json:"Mode"`
	RW          bool   `json:"RW"`
	Propagation string `json:"Propagation"`
}

// Version represents Docker version information
type Version struct {
	Version       string `json:"Version"`
	APIVersion    string `json:"ApiVersion"`
	MinAPIVersion string `json:"MinApiVersion"`
	GitCommit     string `json:"GitCommit"`
	GoVersion     string `json:"GoVersion"`
	OS            string `json:"Os"`
	Arch          string `json:"Arch"`
	BuildTime     string `json:"BuildTime"`
}

// ContainerListOptions represents options for listing containers
type ContainerListOptions struct {
	All    bool
	Limit  int
	Size   bool
	Filter map[string][]string
}

// Client is the interface for Docker operations
type Client interface {
	// Ping checks if the Docker daemon is accessible
	Ping(ctx context.Context) error

	// Version returns Docker version information
	Version(ctx context.Context) (*Version, error)

	// ContainerList returns a list of containers
	ContainerList(ctx context.Context, options ContainerListOptions) ([]Container, error)

	// ContainerLogs returns logs from a container
	ContainerLogs(ctx context.Context, containerID string, follow bool, tail string) (io.ReadCloser, error)
}

// client implements the Client interface
type client struct {
	config     *Config
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Docker client
func NewClient(config *Config) (Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create HTTP client with Unix socket transport
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", config.SocketPath)
			},
			DisableCompression: true,
		},
	}

	baseURL := "http://localhost"

	c := &client{
		config:     config,
		httpClient: httpClient,
		baseURL:    baseURL,
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Ping(ctx); err != nil {
		log.Error("Failed to connect to Docker daemon", "error", err, "socket_path", config.SocketPath)
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	log.Info("Docker client initialized successfully", "socket_path", config.SocketPath)
	return c, nil
}

// doRequest performs an HTTP request to the Docker daemon
func (c *client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	log.Debug("Docker API request", "method", method, "path", path)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("Docker API request failed", "method", method, "path", path, "error", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Error("Docker API error response", "method", method, "path", path, "status", resp.StatusCode, "body", string(bodyBytes))
		return nil, fmt.Errorf("API error: %d %s - %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}

	return resp, nil
}

// Ping checks if the Docker daemon is accessible
func (c *client) Ping(ctx context.Context) error {
	resp, err := c.doRequest(ctx, "GET", "/_ping", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Version returns Docker version information
func (c *client) Version(ctx context.Context) (*Version, error) {
	resp, err := c.doRequest(ctx, "GET", "/version", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var version Version
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return nil, fmt.Errorf("failed to decode version response: %w", err)
	}

	return &version, nil
}

// ContainerList returns a list of containers
func (c *client) ContainerList(ctx context.Context, options ContainerListOptions) ([]Container, error) {
	path := "/containers/json"

	// Build query parameters
	var params []string
	if options.All {
		params = append(params, "all=true")
	}
	if options.Limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", options.Limit))
	}
	if options.Size {
		params = append(params, "size=true")
	}

	// Add filters
	if len(options.Filter) > 0 {
		filters := make(map[string][]string)
		for key, values := range options.Filter {
			filters[key] = values
		}
		filterJSON, _ := json.Marshal(filters)
		params = append(params, fmt.Sprintf("filters=%s", string(filterJSON)))
	}

	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var containers []Container
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("failed to decode containers response: %w", err)
	}

	return containers, nil
}

// ContainerLogs returns logs from a container
func (c *client) ContainerLogs(ctx context.Context, containerID string, follow bool, tail string) (io.ReadCloser, error) {
	path := fmt.Sprintf("/containers/%s/logs", containerID)

	var params []string
	params = append(params, "stdout=true", "stderr=true")

	if follow {
		params = append(params, "follow=true")
	}

	if tail != "" {
		params = append(params, fmt.Sprintf("tail=%s", tail))
	}

	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	// Docker logs can be multiplexed (stdout/stderr combined)
	// We need to handle the stream format
	return &logReader{
		reader: resp.Body,
		closer: resp.Body,
	}, nil
}

// logReader handles Docker log stream format
type logReader struct {
	reader io.Reader
	closer io.Closer
}

func (lr *logReader) Read(p []byte) (n int, err error) {
	return lr.reader.Read(p)
}

func (lr *logReader) Close() error {
	return lr.closer.Close()
}
