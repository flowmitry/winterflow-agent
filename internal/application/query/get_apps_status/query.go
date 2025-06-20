package get_apps_status

import (
	"winterflow-agent/internal/infra/winterflow/grpc/pb"
)

// GetAppsStatusQuery represents a query to retrieve application statuses
type GetAppsStatusQuery struct {
	Request *pb.GetAppsStatusRequestV1
}

// Name returns the name of the query
func (q GetAppsStatusQuery) Name() string {
	return "GetAppsStatus"
}
