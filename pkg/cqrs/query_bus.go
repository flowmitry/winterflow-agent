package cqrs

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

// ErrQueryBusShuttingDown is returned when a query is dispatched to a bus that is shutting down.
var ErrQueryBusShuttingDown = errors.New("query bus is shutting down")

// DefaultQueryBus is a simple implementation of the QueryBus interface.
type DefaultQueryBus struct {
	*Bus
}

// NewQueryBus creates a new DefaultQueryBus.
func NewQueryBus(ctx context.Context) *DefaultQueryBus {
	b := &DefaultQueryBus{
		Bus: NewBus("query"),
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

// validateQueryHandler checks if the handler implements QueryHandler[Q, R] and returns the query name.
func validateQueryHandler(handler interface{}, queryType reflect.Type) (string, error) {
	// Check if the query type implements Query
	queryInstance := reflect.New(queryType).Elem().Interface()
	query, ok := queryInstance.(Query)
	if !ok {
		return "", fmt.Errorf("parameter type %s does not implement Query interface", queryType)
	}

	// Get the handler type and check its Handle method
	handlerType := reflect.TypeOf(handler)
	handleMethod, _ := handlerType.MethodByName("Handle")
	methodType := handleMethod.Type

	// Check that Handle returns two values (result and error)
	if methodType.NumOut() != 2 {
		return "", fmt.Errorf("Handle method must return exactly two values (result and error)")
	}

	// Return the query name
	return query.Name(), nil
}

// Register registers a query handler for a specific query type.
// The handler must implement QueryHandler[Q, R] where Q is a Query type and R is the result type.
func (b *DefaultQueryBus) Register(handler interface{}) error {
	// Find the Handle method
	handlerType := reflect.TypeOf(handler)
	handleMethod, exists := handlerType.MethodByName("Handle")
	if !exists {
		return fmt.Errorf("handler %T does not implement Handle method", handler)
	}

	// Get the query type
	queryType := handleMethod.Type.In(1)

	return b.Bus.Register(handler, queryType, validateQueryHandler)
}

// Dispatch sends a query to its appropriate handler and returns the result.
func (b *DefaultQueryBus) Dispatch(query Query) (interface{}, error) {
	if b.IsShuttingDown() {
		return nil, ErrQueryBusShuttingDown
	}

	handler, exists := b.GetHandler(query.Name())
	if !exists {
		return nil, fmt.Errorf("no handler registered for query %s", query.Name())
	}

	// Increment the active queries counter
	b.IncrementActiveCount()
	defer b.DecrementActiveCount()

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
