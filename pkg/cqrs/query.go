package cqrs

// Query represents a request for information that does not change the state of the system.
// Queries are named with verbs in present tense (e.g., "GetUser").
type Query interface {
	// QueryName returns the name of the query.
	QueryName() string
}

// QueryHandler defines the interface for handling queries.
type QueryHandler[Q Query, R any] interface {
	// Handle executes the query and returns the result or an error.
	Handle(query Q) (R, error)
}

// QueryBus is responsible for dispatching queries to their handlers.
type QueryBus interface {
	// Dispatch sends a query to its appropriate handler and returns the result.
	Dispatch(query Query) (interface{}, error)

	// Register registers a query handler for a specific query type.
	Register(handler interface{}) error

	// Shutdown initiates a graceful shutdown of the query bus.
	// New queries will be rejected, but existing queries will be allowed to complete.
	Shutdown()

	// WaitForCompletion waits for all active queries to complete.
	// This should be called after Shutdown to ensure all queries have finished processing.
	WaitForCompletion()
}
