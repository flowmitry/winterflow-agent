package cqrs

import (
	"fmt"
	"reflect"
	"sync"
)

// DefaultCommandBus is a simple implementation of the CommandBus interface.
type DefaultCommandBus struct {
	handlers map[string]interface{}
	mutex    sync.RWMutex
}

// NewCommandBus creates a new DefaultCommandBus.
func NewCommandBus() *DefaultCommandBus {
	return &DefaultCommandBus{
		handlers: make(map[string]interface{}),
	}
}

// Register registers a command handler for a specific command type.
// The handler must implement CommandHandler[C] where C is a Command type.
func (b *DefaultCommandBus) Register(handler interface{}) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Ptr {
		return fmt.Errorf("handler must be a pointer to a struct, got %T", handler)
	}

	// Check if the handler implements CommandHandler[C]
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
	if methodType.NumIn() != 2 { // receiver + command
		return fmt.Errorf("Handle method must have exactly one parameter (the command)")
	}

	// Get the command type
	cmdType := methodType.In(1)

	// Check if the command type implements Command
	cmdInstance := reflect.New(cmdType).Elem().Interface()
	cmd, ok := cmdInstance.(Command)
	if !ok {
		return fmt.Errorf("parameter type %s does not implement Command interface", cmdType)
	}

	// Register the handler with the command name
	cmdName := cmd.CommandName()
	if _, exists := b.handlers[cmdName]; exists {
		return fmt.Errorf("handler for command %s already registered", cmdName)
	}

	b.handlers[cmdName] = handler
	return nil
}

// Dispatch sends a command to its appropriate handler.
func (b *DefaultCommandBus) Dispatch(cmd Command) error {
	b.mutex.RLock()
	handler, exists := b.handlers[cmd.CommandName()]
	b.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for command %s", cmd.CommandName())
	}

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
