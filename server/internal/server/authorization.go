package server

import (
	"context"
	"fmt"

	v1 "github.com/llm-operator/rbac-manager/api/v1"
	"github.com/llm-operator/rbac-manager/server/internal/cache"
	"github.com/llm-operator/user-manager/pkg/role"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Authorize authorizes the given token and scope.
func (s *Server) Authorize(ctx context.Context, req *v1.AuthorizeRequest) (*v1.AuthorizeResponse, error) {
	if req.Token == "" {
		return nil, status.Errorf(codes.InvalidArgument, "token is required")
	}
	if req.AccessResource == "" {
		return nil, status.Errorf(codes.InvalidArgument, "access resource is required")
	}
	if req.Capability == "" {
		return nil, status.Errorf(codes.InvalidArgument, "capability is required")
	}

	// Check if the token is the API key.
	key, ok := s.cache.GetAPIKeyBySecret(req.Token)
	if ok {
		return &v1.AuthorizeResponse{
			Authorized:   s.authorizedAPIKey(key, toScope(req)),
			User:         &v1.User{Id: key.UserID},
			Organization: &v1.Organization{Id: key.OrganizationID},
			// TODO(kenji): Fill Project.
		}, nil
	}

	is, err := s.tokenIntrospector.TokenIntrospect(req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to introspect token: %v", err)
	}

	if !is.Active {
		return &v1.AuthorizeResponse{Authorized: false}, nil
	}

	user := &v1.User{Id: is.Extra.Email}
	orgs, ok := s.cache.GetOrganizationsByUserID(user.Id)
	if !ok {
		return &v1.AuthorizeResponse{Authorized: false}, nil
	}

	var org *cache.O
	if req.OrganizationId == "" {
		// TODO(aya): handle multiple organizations.
		org = &orgs[0]
	} else {
		// Check if the organization specified in the request is a member of the user.
		for _, o := range orgs {
			if o.OrganizationID == req.OrganizationId {
				org = &o
				break
			}
		}
		if org == nil {
			return &v1.AuthorizeResponse{Authorized: false}, nil
		}
	}

	// TODO(kenji): Validate if the project is a member of the organization.
	role, found := role.OrganizationRoleToString(org.Role)
	if !found {
		return nil, status.Errorf(codes.Internal, "invalid role: %q", org.Role)
	}
	r := &v1.AuthorizeResponse{
		Authorized: s.authorized(role, toScope(req)),
		User:       user,
		Organization: &v1.Organization{
			Id: org.OrganizationID,
		},
		Project: &v1.Project{Id: req.ProjectId},
	}

	return r, nil
}

func (s *Server) authorized(role string, requestScope string) bool {
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

func (s *Server) authorizedAPIKey(apiKey *cache.K, requestScope string) bool {
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

func toScope(req *v1.AuthorizeRequest) string {
	return fmt.Sprintf("%s.%s", req.AccessResource, req.Capability)
}
