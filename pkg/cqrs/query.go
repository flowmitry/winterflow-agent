package cqrs

// Query represents a request for information that does not change the state of the system.
// Queries are named with verbs in present tense (e.g., "GetUser").
type Query interface {
	NameProvider
}

// QueryHandler defines the interface for handling queries.
type QueryHandler[Q Query, R any] interface {
	// Handle executes the query and returns the result or an error.
	Handle(query Q) (R, error)
}

// QueryBus is responsible for dispatching queries to their handlers.
type QueryBus interface {
	ActionProvider

	// Dispatch sends a query to its appropriate handler and returns the result.
	Dispatch(query Query) (interface{}, error)
}
