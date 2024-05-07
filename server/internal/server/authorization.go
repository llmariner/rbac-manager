package server

import (
	"context"

	v1 "github.com/llm-operator/rbac-manager/api/v1"
	"github.com/llm-operator/rbac-manager/server/internal/apikey"
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

	// Check if the token is the API key.
	key, ok := s.apiKeyCache.GetAPIKey(req.Token)
	if ok {
		return &v1.AuthorizeResponse{
			Authorized: s.authorizedAPIKey(key, req.Scope),
		}, nil
	}

	is, err := s.tokenIntrospector.TokenIntrospect(req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to introspect token: %v", err)
	}

	if !is.Active {
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

func (s *Server) authorizedAPIKey(apiKey *apikey.K, requestScope string) bool {
	allowedScopes, ok := s.roleScopesMapper[apiKey.Role]
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
