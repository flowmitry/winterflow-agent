# Winterflow Agent Development Guidelines

## Architecture

The Winterflow Agent follows a clean architecture approach with the following layers:

1. **Domain Layer** (`internal/domain/`):
   - Contains the core business logic and domain models
   - Independent of any external frameworks or infrastructure
   - Defines repository interfaces that are implemented by the infrastructure layer

2. **Application Layer** (`internal/application/`):
   - Contains application-specific logic
   - Implements use cases using domain models and repositories
   - Uses CQRS pattern with commands and queries
   - Must not depend directly on infrastructure, only on domain models

3. **Infrastructure Layer** (`internal/infra/`):
   - Implements repository interfaces defined in the domain layer
   - Contains adapters for external services (gRPC, HTTP, etc.)
   - Handles data persistence and external communication
   - Contains assemblers for transforming between domain and infrastructure models

## Domain-Infrastructure Transformation

For transformations between domain and infrastructure models:

- Use assemblers in the infrastructure layer
- For gRPC client transformations, use `internal/infra/winterflow/grpc/client/assemblers.go`
- Never expose infrastructure models to the application layer
- Never expose domain models directly to external clients
- Use a consistent naming for methods `<model_name>ToProto<proto_model_name>` and
  `Proto<proto_model_name>To<model_name>`

Example:

```go
// In assemblers.go
// Domain to Infrastructure
func AppToProtoAppV1(app *model.App) *pb.AppV1 {
// Transform domain model to protobuf model
}

// Infrastructure to Domain
func ProtoGetAppRequestV1ToGetAppRequest(request *pb.GetAppRequestV1) *model.GetAppRequest {
// Transform protobuf request to domain request
}
```

## Command Implementation Guidelines

When implementing commands:

1. Define domain models in `internal/domain/model/`
2. Define command struct in application layer using domain models
3. Pass properties directly to the Commands, do not create Request models
4. Implement command handler that uses domain models
5. Use direct transformation in infrastructure layer to convert between domain and infrastructure models

Example:

```go
// Command using domain models
type SaveAppCommand struct {
AppID     string
Config    *model.AppConfig
Variables model.VariableMap
Files     map[string][]byte
}

// Handler using domain models
func (h *SaveAppHandler) Handle(cmd SaveAppCommand) error {
// Implementation using domain models
}

// Infrastructure layer using direct transformation
func HandleSaveAppRequest(commandBus cqrs.CommandBus, saveAppRequest *pb.SaveAppRequestV1, agentID string) (*pb.AgentMessage, error) {
// Convert variables to VariableMap
variables := make(model.VariableMap)
for _, v := range saveAppRequest.App.Variables {
variables[v.Id] = string(v.Content)
}

// Convert files to map[string][]byte
files := make(map[string][]byte)
for _, file := range saveAppRequest.App.Files {
files[file.Id] = file.Content
}

// Parse config bytes into AppConfig
appConfig, err := model.ParseAppConfig(saveAppRequest.App.Config)
if err != nil {
log.Error("Error parsing app config: %v", err)
appConfig = &model.AppConfig{ID: saveAppRequest.App.AppId}
}

// Create and dispatch the command
cmd := SaveAppCommand{
AppID:     saveAppRequest.App.AppId,
Config:    appConfig,
Variables: variables,
Files:     files,
}

// Dispatch the command to the handler
err := commandBus.Dispatch(cmd)

// Handle the response
// ...
}
```

## Query Implementation Guidelines

When implementing queries:

1. Define domain models in `internal/domain/model/`
2. Define query struct in application layer using domain models
3. Pass properties directly to the Queries, do not create Request models
4. Implement query handler that returns domain models
5. Use assemblers in infrastructure layer to transform between domain and infrastructure models

Example:

```go
// Query using domain models
type GetAppQuery struct {
AppID      string
AppVersion uint32
}

// Handler returning domain models
func (h *GetAppQueryHandler) Handle(query GetAppQuery) (*model.App, error) {
// Implementation using domain models
}

// Infrastructure layer using assemblers
func HandleGetAppQuery(queryBus cqrs.QueryBus, getAppRequest *pb.GetAppRequestV1) {
// Convert request to domain model
domainRequest := FromProtoGetAppRequest(getAppRequest)

// Create and dispatch query
query := GetAppQuery{
AppID: getAppRequest.appID,
AppVersion: getAppRequest.version
}
result, err := queryBus.Dispatch(query)

// Convert result back to infrastructure model
domainApp := result.(*model.App)
protoApp := ToProtoApp(domainApp)
}
```

## Technical Stack

- Backend: Go 1.24.0
- Use only standard library for GO until I asked overwise.
- Use GRPC.
- Use JSON file for the configuration storage.

## Logging

- Use custom logging package (pkg/log) for all logging
- Import as: `log "winterflow-agent/pkg/log"`
- Use structured logging methods: Debug, Info, Warn, Error
- Use Printf for compatibility with existing code
- Use Fatalf/Fatal for fatal errors
- All logs are JSON formatted and include timestamps
- Log levels: Debug, Info, Warn, Error
- Include relevant context in log messages
- Use consistent log message format: [LEVEL] message
- Include error details when logging errors
- Use proper log levels for different types of messages

# GO shared libraries and packages:

Create a package in pkg if you need to create some shared component.

# Adding New Capabilities:

To add a new capability to the system, follow these steps:

1. Add a new constant in pkg/capabilities/capability.go:
   ```go
   const (
       CapabilityNewTool = "new-tool"
   )
   ```

2. Create a new file pkg/capabilities/new_tool.go:
   ```go
   package capabilities

   import (
       "os/exec"
       "strings"
   )

   // NewToolCapability represents the New Tool capability
   type NewToolCapability struct {
       version string
   }

   // NewNewToolCapability creates a new New Tool capability
   func NewNewToolCapability() *NewToolCapability {
       return &NewToolCapability{
           version: "1.0", // Default version
       }
   }

   // Name returns the name of the capability
   func (c *NewToolCapability) Name() string {
       return CapabilityNewTool
   }

   // Version returns the version of the capability
   func (c *NewToolCapability) Version() string {
       return c.version
   }

   // IsAvailable checks if New Tool is available on the system
   func (c *NewToolCapability) IsAvailable() bool {
       cmd := exec.Command("new-tool", "--version")
       output, err := cmd.Output()
       if err != nil {
           return false
       }

       // Parse version from output
       versionStr := string(output)
       if strings.Contains(versionStr, "new-tool version") {
           // Extract version from output
           parts := strings.Split(versionStr, " ")
           if len(parts) > 2 {
               c.version = parts[2]
           }
           return true
       }
       return false
   }
   ```

3. Add the new capability to the factory in pkg/capabilities/capability.go:
   ```go
   func NewCapabilityFactory() *CapabilityFactory {
       return &CapabilityFactory{
           capabilities: []Capability{
               // ... existing capabilities ...
               NewNewToolCapability(),
           },
       }
   }
   ```

4. Update the SystemCapabilities struct in internal/agent/capabilities.go:
   ```go
   type SystemCapabilities struct {
       // ... existing fields ...
       NewTool string
   }
   ```

5. Update the ToMap method in internal/agent/capabilities.go:
   ```go
   func (c SystemCapabilities) ToMap() map[string]string {
       return map[string]string{
           // ... existing mappings ...
           "new-tool": c.NewTool,
       }
   }
   ```

6. Update the GetSystemCapabilities function in internal/agent/capabilities.go:
   ```go
   func GetSystemCapabilities() SystemCapabilities {
       // ... existing code ...
       for _, c := range factory.GetAllCapabilities() {
           if c.IsAvailable() {
               switch c.Name() {
               // ... existing cases ...
               case capabilities.CapabilityNewTool:
                   result.NewTool = c.Version()
               }
           }
       }
       return result
   }
   ```

Best Practices:

- Use consistent naming conventions (e.g., snake_case for file names, CamelCase for types)
- Include proper error handling in IsAvailable()
- Add appropriate comments and documentation
- Test the capability detection on different platforms
- Keep version parsing logic consistent with existing capabilities
- Use meaningful default versions
- Follow the existing code structure and patterns

# Command line tools

- Use makefile for build, run, test, lint, format, etc.

# Tasks completion

After each task run `make build` and fix errors if any.
