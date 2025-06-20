package get_app

// GetAppQuery represents a query to retrieve an application
type GetAppQuery struct {
	AppID      string
	AppVersion uint32
}

// Name returns the name of the query
func (q GetAppQuery) Name() string {
	return "GetApp"
}
