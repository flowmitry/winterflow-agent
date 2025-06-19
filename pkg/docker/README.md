# Docker Client Package

This package provides a simple Docker client implementation using only Go's standard library. It communicates directly
with the Docker daemon via Unix socket.

## Features

- **Zero external dependencies** - uses only Go standard library
- **Cross-platform support** - works on Linux, macOS, and Windows
- **Unix socket communication** - direct connection to Docker daemon
- **Structured logging** - integrates with the project's logging system
- **Thread-safe** - safe for concurrent use

## Supported Operations

The client supports the following core Docker operations:

- `Ping` - Check Docker daemon connectivity
- `Version` - Get Docker version information
- `ContainerList` - List containers with filtering options
- `ContainerLogs` - Stream container logs

## Usage

### Basic Usage

```go
import (
    "context"
    "log"
    "winterflow-agent/pkg/docker"
)

func main() {
    // Create client with default configuration
    client, err := docker.NewClient(nil)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Check connectivity
    if err := client.Ping(ctx); err != nil {
        log.Fatal("Docker daemon not accessible:", err)
    }

    // Get Docker version
    version, err := client.Version(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Docker version: %s\n", version.Version)

    // List all containers
    containers, err := client.ContainerList(ctx, docker.ContainerListOptions{
        All: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, container := range containers {
        fmt.Printf("Container: %s (%s)\n", container.Names[0], container.Status)
    }
}
```

### Custom Configuration

```go
config := &docker.Config{
    SocketPath: "/var/run/docker.sock",
    Timeout:    30 * time.Second,
}

client, err := docker.NewClient(config)
if err != nil {
    log.Fatal(err)
}
```

### Container Listing with Filters

```go
// List only running containers
containers, err := client.ContainerList(ctx, docker.ContainerListOptions{
    Filter: map[string][]string{
        "status": {"running"},
    },
})

// List containers with specific labels
containers, err := client.ContainerList(ctx, docker.ContainerListOptions{
    Filter: map[string][]string{
        "label": {"com.docker.compose.project=myapp"},
    },
})
```

### Container Logs

```go
// Get container logs
logReader, err := client.ContainerLogs(ctx, containerID, false, "100")
if err != nil {
    log.Fatal(err)
}
defer logReader.Close()

// Read logs
buf := make([]byte, 1024)
for {
    n, err := logReader.Read(buf)
    if err != nil {
        if err == io.EOF {
            break
        }
        log.Fatal(err)
    }
    fmt.Print(string(buf[:n]))
}
```

## Configuration

The client supports the following configuration options:

- `SocketPath`: Path to Docker daemon socket (default: `/var/run/docker.sock` on Unix, `\\.\pipe\docker_engine` on
  Windows)
- `Timeout`: HTTP request timeout (default: 30 seconds)

## Platform Support

### Linux/macOS

- Default socket path: `/var/run/docker.sock`
- Requires Docker daemon to be running
- User must have permissions to access the socket

### Windows

- Default socket path: `\\.\pipe\docker_engine`
- Requires Docker Desktop to be running
- Works with both Windows containers and Linux containers via Docker Desktop

## Error Handling

The client provides structured errors with context:

```go
if err := client.Ping(ctx); err != nil {
    // Handle connection errors
    log.Error("Docker daemon connection failed", "error", err)
    return
}

containers, err := client.ContainerList(ctx, options)
if err != nil {
    // Handle API errors
    log.Error("Failed to list containers", "error", err)
    return
}
```

## Thread Safety

The client is thread-safe and can be used concurrently from multiple goroutines. Each method call is independent and
doesn't affect the client's internal state.

## API Compatibility

This client uses Docker's default API version negotiation, making it compatible with most Docker installations. It has
been tested with:

- Docker Engine 20.10+
- Docker Desktop for Windows/Mac
- Docker CE/EE

The client will automatically use the highest API version supported by both the client and the Docker daemon. 