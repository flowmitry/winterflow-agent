package get_apps_status

// GetAppsStatusQuery represents a query to retrieve application statuses
type GetAppsStatusQuery struct {
	// No fields needed for this query
}

// Name returns the name of the query
func (q GetAppsStatusQuery) Name() string {
	return "GetAppsStatus"
}
