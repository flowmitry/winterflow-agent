package cqrs

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

// ErrCommandBusShuttingDown is returned when a command is dispatched to a bus that is shutting down.
var ErrCommandBusShuttingDown = errors.New("command bus is shutting down")

// DefaultCommandBus is a simple implementation of the CommandBus interface.
type DefaultCommandBus struct {
	*Bus
}

// NewCommandBus creates a new DefaultCommandBus.
func NewCommandBus(ctx context.Context) *DefaultCommandBus {
	b := &DefaultCommandBus{
		Bus: NewBus("command"),
	}

	// Listen for context cancellation and initiate graceful shutdown.
	if ctx != nil {
		go func() {
			<-ctx.Done()
			b.Shutdown()
		}()
	}

	return b
}

// validateCommandHandler checks if the handler implements CommandHandler[C] and returns the command name.
func validateCommandHandler(handler interface{}, cmdType reflect.Type) (string, error) {
	// Check if the command type implements Command
	cmdInstance := reflect.New(cmdType).Elem().Interface()
	cmd, ok := cmdInstance.(Command)
	if !ok {
		return "", fmt.Errorf("parameter type %s does not implement Command interface", cmdType)
	}

	// Return the command name
	return cmd.Name(), nil
}

// Register registers a command handler for a specific command type.
// The handler must implement CommandHandler[C] where C is a Command type.
func (b *DefaultCommandBus) Register(handler interface{}) error {
	// Find the Handle method
	handlerType := reflect.TypeOf(handler)
	handleMethod, exists := handlerType.MethodByName("Handle")
	if !exists {
		return fmt.Errorf("handler %T does not implement Handle method", handler)
	}

	// Get the command type
	cmdType := handleMethod.Type.In(1)

	return b.Bus.Register(handler, cmdType, validateCommandHandler)
}

// Dispatch sends a command to its appropriate handler.
func (b *DefaultCommandBus) Dispatch(cmd Command) error {
	if b.IsShuttingDown() {
		return ErrCommandBusShuttingDown
	}

	handler, exists := b.GetHandler(cmd.Name())
	if !exists {
		return fmt.Errorf("no handler registered for command %s", cmd.Name())
	}

	// Increment the active commands counter
	b.IncrementActiveCount()
	defer b.DecrementActiveCount()

	// Call the handler's Handle method with the command
	handlerValue := reflect.ValueOf(handler)
	handleMethod := handlerValue.MethodByName("Handle")

	results := handleMethod.Call([]reflect.Value{reflect.ValueOf(cmd)})

	// Check for error
	if len(results) > 0 && !results[0].IsNil() {
		return results[0].Interface().(error)
	}

	return nil
}
