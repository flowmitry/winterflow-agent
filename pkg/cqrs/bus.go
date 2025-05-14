// Package cqrs implements the Command Query Responsibility Segregation pattern.
package cqrs

import (
	"fmt"
	"reflect"
	"sync"
)

// NameProvider is an interface for both Command and Query types
// that provides a way to get the name of the message.
type NameProvider interface {
	// Name returns the name of the message (command or query).
	Name() string
}

// ActionProvider defines an interface for managing command handlers and controlling their lifecycle.
// Register adds a command handler for a specific command type and returns an error if registration fails.
// Shutdown triggers a graceful termination of the command bus, disallowing new commands while letting current ones finish.
// WaitForCompletion ensures that all active commands are processed to completion after a shutdown has been initiated.
type ActionProvider interface {
	// Register registers a command handler for a specific command type.
	Register(handler interface{}) error

	// Shutdown initiates a graceful shutdown of the command bus.
	// New commands will be rejected, but existing commands will be allowed to complete.
	Shutdown()

	// WaitForCompletion waits for all active commands to complete.
	// This should be called after Shutdown to ensure all commands have finished processing.
	WaitForCompletion()
}

// Bus is a generic implementation that can be used by both command and query buses.
type Bus struct {
	handlers       map[string]interface{}
	mutex          sync.RWMutex
	isShuttingDown bool
	activeMessages sync.WaitGroup
	busType        string // "command" or "query"
}

// NewBus creates a new Bus with the specified type.
func NewBus(busType string) *Bus {
	return &Bus{
		handlers: make(map[string]interface{}),
		busType:  busType,
	}
}

// Register registers a handler for a specific message type.
// The handler must implement the appropriate interface (CommandHandler or QueryHandler).
func (b *Bus) Register(handler interface{}, messageType reflect.Type, validateFunc func(interface{}, reflect.Type) (string, error)) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Ptr {
		return fmt.Errorf("handler must be a pointer to a struct, got %T", handler)
	}

	// Check if the handler implements required methods
	if handlerType.NumMethod() == 0 {
		return fmt.Errorf("handler %T does not implement any methods", handler)
	}

	// Find the Handle method
	handleMethod, exists := handlerType.MethodByName("Handle")
	if !exists {
		return fmt.Errorf("handler %T does not implement Handle method", handler)
	}

	// Check method signature
	methodType := handleMethod.Type
	if methodType.NumIn() != 2 { // receiver + message
		return fmt.Errorf("Handle method must have exactly one parameter (the %s)", b.busType)
	}

	// Validate the message type and get its name
	messageName, err := validateFunc(handler, messageType)
	if err != nil {
		return err
	}

	// Register the handler with the message name
	if _, exists := b.handlers[messageName]; exists {
		return fmt.Errorf("handler for %s %s already registered", b.busType, messageName)
	}

	b.handlers[messageName] = handler
	return nil
}

// Shutdown initiates a graceful shutdown of the bus.
// New messages will be rejected, but existing messages will be allowed to complete.
func (b *Bus) Shutdown() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.isShuttingDown = true
}

// WaitForCompletion waits for all active messages to complete.
// This should be called after Shutdown to ensure all messages have finished processing.
func (b *Bus) WaitForCompletion() {
	b.activeMessages.Wait()
}

// IsShuttingDown returns true if the bus is shutting down.
func (b *Bus) IsShuttingDown() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.isShuttingDown
}

// GetHandler returns the handler for the given message name.
func (b *Bus) GetHandler(messageName string) (interface{}, bool) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	handler, exists := b.handlers[messageName]
	return handler, exists
}

// IncrementActiveCount increments the active message counter.
func (b *Bus) IncrementActiveCount() {
	b.activeMessages.Add(1)
}

// DecrementActiveCount decrements the active message counter.
func (b *Bus) DecrementActiveCount() {
	b.activeMessages.Done()
}
