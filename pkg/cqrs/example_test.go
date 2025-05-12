package cqrs_test

import (
	"fmt"
	"winterflow-agent/pkg/cqrs"
)

// Example command
type CreateUserCommand struct {
	Username string
	Email    string
}

func (c CreateUserCommand) CommandName() string {
	return "CreateUser"
}

// Example command handler
type CreateUserHandler struct {
	// In a real application, this would have dependencies like a repository
}

func (h *CreateUserHandler) Handle(cmd CreateUserCommand) error {
	// In a real application, this would create a user in a database
	fmt.Printf("Creating user: %s (%s)\n", cmd.Username, cmd.Email)
	return nil
}

// Example query
type GetUserQuery struct {
	UserID string
}

func (q GetUserQuery) QueryName() string {
	return "GetUser"
}

// Example user model
type User struct {
	ID       string
	Username string
	Email    string
}

// Example query handler
type GetUserHandler struct {
	// In a real application, this would have dependencies like a repository
}

func (h *GetUserHandler) Handle(query GetUserQuery) (User, error) {
	// In a real application, this would fetch a user from a database
	return User{
		ID:       query.UserID,
		Username: "example_user",
		Email:    "user@example.com",
	}, nil
}

// ExampleCommandBus demonstrates how to use the command bus
func Example_commandBus() {
	// Create a new command bus
	commandBus := cqrs.NewCommandBus()

	// Register a command handler
	handler := &CreateUserHandler{}
	err := commandBus.Register(handler)
	if err != nil {
		fmt.Printf("Error registering handler: %v\n", err)
		return
	}

	// Create and dispatch a command
	cmd := CreateUserCommand{
		Username: "john_doe",
		Email:    "john@example.com",
	}

	err = commandBus.Dispatch(cmd)
	if err != nil {
		fmt.Printf("Error dispatching command: %v\n", err)
		return
	}

	// Output:
	// Creating user: john_doe (john@example.com)
}

// ExampleQueryBus demonstrates how to use the query bus
func Example_queryBus() {
	// Create a new query bus
	queryBus := cqrs.NewQueryBus()

	// Register a query handler
	handler := &GetUserHandler{}
	err := queryBus.Register(handler)
	if err != nil {
		fmt.Printf("Error registering handler: %v\n", err)
		return
	}

	// Create and dispatch a query
	query := GetUserQuery{
		UserID: "user123",
	}

	result, err := queryBus.Dispatch(query)
	if err != nil {
		fmt.Printf("Error dispatching query: %v\n", err)
		return
	}

	// Type assertion to get the specific result type
	user, ok := result.(User)
	if !ok {
		fmt.Println("Error: result is not a User")
		return
	}

	fmt.Printf("Found user: %s (%s)\n", user.Username, user.Email)

	// Output:
	// Found user: example_user (user@example.com)
}
