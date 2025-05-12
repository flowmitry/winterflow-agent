package cqrs

import (
	"fmt"
	"reflect"
	"sync"
)

// DefaultQueryBus is a simple implementation of the QueryBus interface.
type DefaultQueryBus struct {
	handlers map[string]interface{}
	mutex    sync.RWMutex
}

// NewQueryBus creates a new DefaultQueryBus.
func NewQueryBus() *DefaultQueryBus {
	return &DefaultQueryBus{
		handlers: make(map[string]interface{}),
	}
}

// Register registers a query handler for a specific query type.
// The handler must implement QueryHandler[Q, R] where Q is a Query type and R is the result type.
func (b *DefaultQueryBus) Register(handler interface{}) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Ptr {
		return fmt.Errorf("handler must be a pointer to a struct, got %T", handler)
	}

	// Check if the handler implements QueryHandler[Q, R]
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
	if methodType.NumIn() != 2 { // receiver + query
		return fmt.Errorf("Handle method must have exactly one parameter (the query)")
	}

	if methodType.NumOut() != 2 { // result + error
		return fmt.Errorf("Handle method must return exactly two values (result and error)")
	}

	// Get the query type
	queryType := methodType.In(1)

	// Check if the query type implements Query
	queryInstance := reflect.New(queryType).Elem().Interface()
	query, ok := queryInstance.(Query)
	if !ok {
		return fmt.Errorf("parameter type %s does not implement Query interface", queryType)
	}

	// Register the handler with the query name
	queryName := query.QueryName()
	if _, exists := b.handlers[queryName]; exists {
		return fmt.Errorf("handler for query %s already registered", queryName)
	}

	b.handlers[queryName] = handler
	return nil
}

// Dispatch sends a query to its appropriate handler and returns the result.
func (b *DefaultQueryBus) Dispatch(query Query) (interface{}, error) {
	b.mutex.RLock()
	handler, exists := b.handlers[query.QueryName()]
	b.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no handler registered for query %s", query.QueryName())
	}

	// Call the handler's Handle method with the query
	handlerValue := reflect.ValueOf(handler)
	handleMethod := handlerValue.MethodByName("Handle")

	results := handleMethod.Call([]reflect.Value{reflect.ValueOf(query)})

	// Check for error (second return value)
	if !results[1].IsNil() {
		return nil, results[1].Interface().(error)
	}

	// Return the result (first return value)
	return results[0].Interface(), nil
}
