package get_app_logs

// GetAppLogsQuery represents a query to retrieve logs for an application in a given time range.
// When Since or Until is zero, the boundary is ignored (i.e. retrieve from the beginning or up to now).
// All timestamps are Unix seconds.
type GetAppLogsQuery struct {
	AppID string
	Since int64
	Until int64
}

// Name returns the name of the query.
func (q GetAppLogsQuery) Name() string {
	return "GetAppLogs"
}
