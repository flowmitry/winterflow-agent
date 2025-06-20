package get_app

import (
	"winterflow-agent/internal/infra/winterflow/grpc/pb"
)

// GetAppQuery represents a query to retrieve an application
type GetAppQuery struct {
	Request *pb.GetAppRequestV1
}

// Name returns the name of the query
func (q GetAppQuery) Name() string {
	return "GetApp"
}
