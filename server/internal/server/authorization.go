package server

import (
	"context"

	v1 "github.com/llm-operator/rbac-manager/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Authorize authorizes the given token and scope.
func (s *Server) Authorize(ctx context.Context, req *v1.AuthorizeRequest) (*v1.AuthorizeResponse, error) {
	if req.Token == "" {
		return nil, status.Errorf(codes.InvalidArgument, "token is required")
	}
	if req.Scope == "" {
		return nil, status.Errorf(codes.InvalidArgument, "scope is required")
	}

	is, err := s.dexClient.TokenIntrospect(req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to introspect token: %v", err)
	} else if !is.Active {
		return &v1.AuthorizeResponse{Authorized: false}, nil
	}

	return &v1.AuthorizeResponse{
		Authorized: s.authorized(is.Extra.Email, req.Scope),
	}, nil
}

func (s *Server) authorized(user, requestScope string) bool {
	org, ok := s.userOrgMapper[user]
	if !ok {
		return false
	}
	role, ok := s.orgRoleMapper[org]
	if !ok {
		return false
	}
	allowedScopes, ok := s.roleScopesMapper[role]
	if !ok {
		return false
	}
	for _, s := range allowedScopes {
		if s == requestScope {
			return true
		}
	}
	return false
}
