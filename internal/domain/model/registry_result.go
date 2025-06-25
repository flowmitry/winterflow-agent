package model

// GetRegistriesResult represents the result of fetching all Docker registries configured on the host.
// Keeping this struct in the domain layer allows application/query handlers to stay independent from
// infrastructure details while still returning rich data structures when required.
type GetRegistriesResult struct {
	Registries []Registry
}
