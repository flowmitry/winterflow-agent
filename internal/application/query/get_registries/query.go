package get_registries

// GetRegistriesQuery represents a query to retrieve all configured Docker registries.
// It contains no fields as the operation does not require additional input.
type GetRegistriesQuery struct{}

// Name returns the unique name of the query so that the CQRS bus can route it.
func (q GetRegistriesQuery) Name() string {
	return "GetRegistries"
}
