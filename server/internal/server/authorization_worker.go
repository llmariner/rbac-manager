package server

import (
	"context"

	v1 "github.com/llmariner/rbac-manager/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthorizeWorker authorizes the given token.
func (s *Server) AuthorizeWorker(ctx context.Context, req *v1.AuthorizeWorkerRequest) (*v1.AuthorizeWorkerResponse, error) {
	if req.Token == "" {
		return nil, status.Errorf(codes.InvalidArgument, "token is required")
	}

	c, ok := s.cache.GetClusterByRegistrationKey(req.Token)
	if !ok {
		return &v1.AuthorizeWorkerResponse{
			Authorized: false,
		}, nil
	}

	return &v1.AuthorizeWorkerResponse{
		Authorized: true,
		Cluster:    &v1.Cluster{Id: c.ID},
		TenantId:   c.TenantID,
	}, nil
}
