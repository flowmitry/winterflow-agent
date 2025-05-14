// Package cqrs implements the Command Query Responsibility Segregation pattern.
// This pattern separates read and write operations for a data store, allowing
// for better scalability, performance, and maintainability.
package cqrs

// Command represents a command that changes the state of the system.
// Commands are named with verbs in imperative form (e.g., "CreateUser").
type Command interface {
	// CommandName returns the name of the command.
	CommandName() string
}

// CommandHandler defines the interface for handling commands.
type CommandHandler[C Command] interface {
	// Handle executes the command and returns an error if the command fails.
	Handle(cmd C) error
}

// CommandBus is responsible for dispatching commands to their handlers.
type CommandBus interface {
	// Dispatch sends a command to its appropriate handler.
	Dispatch(cmd Command) error

	// Register registers a command handler for a specific command type.
	Register(handler interface{}) error

	// Shutdown initiates a graceful shutdown of the command bus.
	// New commands will be rejected, but existing commands will be allowed to complete.
	Shutdown()

	// WaitForCompletion waits for all active commands to complete.
	// This should be called after Shutdown to ensure all commands have finished processing.
	WaitForCompletion()
}
