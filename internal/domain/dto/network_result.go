package dto

import "winterflow-agent/internal/domain/model"

// GetNetworksResult represents the result of fetching available Docker networks on the host.
// Placing this struct in the domain layer allows application/query handlers to remain
// independent from infrastructure details while still returning rich data structures.
type GetNetworksResult struct {
	Networks []model.Network
}
