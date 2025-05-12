# CQRS Package

This package implements the Command Query Responsibility Segregation (CQRS) pattern for Go applications.

## Overview

CQRS is an architectural pattern that separates read and write operations for a data store, allowing for better
scalability, performance, and maintainability. This package provides the core components needed to implement CQRS in
your application:

- Commands: Represent actions that change the state of the system
- Queries: Represent requests for information that do not change state
- Command Handlers: Process specific command types
- Query Handlers: Process specific query types and return results
- Command Bus: Dispatches commands to their appropriate handlers
- Query Bus: Dispatches queries to their appropriate handlers

## Usage

### Commands

Create a command by implementing the `Command` interface:

```
// Example command
type CreateUserCommand struct {
    Username string
    Email    string
}

func (c CreateUserCommand) CommandName() string {
    return "CreateUser"
}
```

### Command Handlers

Create a command handler by implementing the `CommandHandler` interface:

```
// Example command handler
type CreateUserHandler struct {
    // Dependencies like repositories would go here
}

func (h *CreateUserHandler) Handle(cmd CreateUserCommand) error {
    // Implementation to create a user
    return nil
}
```

### Queries

Create a query by implementing the `Query` interface:

```
// Example query
type GetUserQuery struct {
    UserID string
}

func (q GetUserQuery) QueryName() string {
    return "GetUser"
}
```

### Query Handlers

Create a query handler by implementing the `QueryHandler` interface:

```
// Example user model
type User struct {
    ID       string
    Username string
    Email    string
}

// Example query handler
type GetUserHandler struct {
    // Dependencies like repositories would go here
}

func (h *GetUserHandler) Handle(query GetUserQuery) (User, error) {
    // Implementation to get a user
    return User{}, nil
}
```

### Using the Command Bus

```
// Example of using the command bus
// Create a new command bus
bus := cqrs.NewCommandBus()

// Register a command handler
handler := &CreateUserHandler{}
err := bus.Register(handler)
if err != nil {
    // Handle error
}

// Create and dispatch a command
cmd := CreateUserCommand{
    Username: "john_doe",
    Email:    "john@example.com",
}

err = bus.Dispatch(cmd)
if err != nil {
    // Handle error
}
```

### Using the Query Bus

```
// Example of using the query bus
// Create a new query bus
bus := cqrs.NewQueryBus()

// Register a query handler
handler := &GetUserHandler{}
err := bus.Register(handler)
if err != nil {
    // Handle error
}

// Create and dispatch a query
query := GetUserQuery{
    UserID: "user123",
}

result, err := bus.Dispatch(query)
if err != nil {
    // Handle error
}

// Type assertion to get the specific result type
user, ok := result.(User)
if !ok {
    // Handle type assertion error
}

// Use the user data
fmt.Printf("User: %s\n", user.Username)
```

## Benefits of CQRS

- **Separation of concerns**: Read and write operations can be optimized independently
- **Scalability**: Read and write operations can be scaled separately
- **Performance**: Queries can be optimized for specific data access patterns
- **Flexibility**: Different data models can be used for reading and writing
- **Maintainability**: Simpler, more focused code for each operation type

## Implementation Details

This package uses reflection to dynamically dispatch commands and queries to their handlers. This allows for a clean API
while maintaining type safety at runtime.
