package get_networks

// GetNetworksQuery represents a query to retrieve all available Docker networks.
// It contains no fields as the operation does not require additional input.
type GetNetworksQuery struct{}

// Name returns the unique name of the query so that the CQRS bus can route it.
func (q GetNetworksQuery) Name() string {
	return "GetNetworks"
}
