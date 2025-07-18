// Package cqrs implements the Command Query Responsibility Segregation pattern.
// This pattern separates read and write operations for a data store, allowing
// for better scalability, performance, and maintainability.
package cqrs

// Command represents a command that changes the state of the system.
// Commands are named with verbs in imperative form (e.g., "CreateUser").
type Command interface {
	NameProvider
}

// CommandHandler defines the interface for handling commands.
type CommandHandler[C Command] interface {
	// Handle executes the command and returns an error if the command fails.
	Handle(cmd C) error
}

// CommandBus is responsible for dispatching commands to their handlers.
type CommandBus interface {
	ActionProvider

	// Dispatch sends a command to its appropriate handler.
	Dispatch(cmd Command) error
}
