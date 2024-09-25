package server

import (
	"context"
	"fmt"
	"strings"

	uv1 "github.com/llm-operator/user-manager/api/v1"
	"github.com/llm-operator/user-manager/pkg/userid"
	v1 "github.com/llmariner/rbac-manager/api/v1"
	"github.com/llmariner/rbac-manager/server/internal/cache"
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
		project, found := s.cache.GetProjectByID(key.ProjectID)
		if !found {
			return &v1.AuthorizeResponse{Authorized: false}, nil
		}

		return &v1.AuthorizeResponse{
			Authorized:   s.authorizedAPIKey(key, toScope(req)),
			User:         &v1.User{Id: key.UserID},
			Organization: &v1.Organization{Id: key.OrganizationID},
			Project: &v1.Project{
				Id:                     key.ProjectID,
				AssignedKubernetesEnvs: s.assignedKubernetesEnvs(project.KubernetesNamespace, key.TenantID),
			},
			TenantId: key.TenantID,
		}, nil
	}

	is, err := s.tokenIntrospector.TokenIntrospect(req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to introspect token: %v", err)
	}

	if !is.Active {
		return &v1.AuthorizeResponse{Authorized: false}, nil
	}

	userID := userid.Normalize(is.Extra.Email)
	u, ok := s.cache.GetUserByID(userID)
	if !ok {
		return &v1.AuthorizeResponse{Authorized: false}, nil
	}

	if strings.HasPrefix(req.AccessResource, "api.organizations") {
		// Do not check further as the resource is not project-scoped, and we cannot tell an associated project.
		// We let the caller perform additional check.
		return &v1.AuthorizeResponse{
			Authorized: true,
			User: &v1.User{
				Id: userID,
			},
			Organization: &v1.Organization{},
			Project:      &v1.Project{},
			TenantId:     u.TenantID,
		}, nil
	}

	pr, err := s.findAssociatedProjectAndRoles(userID, req.OrganizationId, req.ProjectId)
	if err != nil {
		// TODO(kenji): Return a more specific error?
		return &v1.AuthorizeResponse{Authorized: false}, nil
	}

	return &v1.AuthorizeResponse{
		Authorized: s.authorized(toScope(req), pr.orgRole, pr.projectRole),
		User: &v1.User{
			Id: userID,
		},
		Organization: &v1.Organization{
			Id: pr.project.OrganizationID,
		},
		Project: &v1.Project{
			Id:                     pr.project.ID,
			AssignedKubernetesEnvs: s.assignedKubernetesEnvs(pr.project.KubernetesNamespace, u.TenantID),
		},
		TenantId: u.TenantID,
	}, nil
}

func (s *Server) authorized(
	requestScope string,
	orgRole uv1.OrganizationRole,
	projectRole uv1.ProjectRole,
) bool {
	// TODO(kenji): Implement the logic based on https://help.openai.com/en/articles/9186755-managing-your-work-in-the-api-platform-with-projects.
	// Here is a snippet from the document:
	//
	//
	// Role: Owner, Scope: Organization
	// - Can create/view all projects, all users, all API keys.
	// - Has the ability to monitor across all projects within the organization with the Projects page.
	// - Able to set billing controls.
	// - Can grant permissions to view usage information for others in the org.
	// - Can archive projects.
	//
	// Role: Reader, Scope: Organization
	// - Can perform inference, use resources, and create keys.
	// - Can be added to projects.
	// - Cannot create projects and manage users.
	//
	// Role: Owner, Scope: Project
	// - Can add other users to the project and rename the project, as well as all the abilities of a Member.
	// - Can archive the project.
	//
	// Role: Member, Scope: Project
	// - Can perform inference, use resources, and create keys at the project level.
	//
	//
	// The definition of "Reader" is not very clear, but here is our interpretation:
	//
	// Suppose org O and project P. P belongs to O. A user with one of the following roles can perform inference on P:
	// - A user that has the "owner" role for O.
	// - A user that has the "owner" role for P.
	// - A user that has the "member" role for P.
	//
	// A user with the "reader" role for O cannot perform inference on P unless the user is a project owner or a member.
	var role string
	switch orgRole {
	case uv1.OrganizationRole_ORGANIZATION_ROLE_OWNER:
		role = "organizationOwner"
	case uv1.OrganizationRole_ORGANIZATION_ROLE_READER:
		switch projectRole {
		case uv1.ProjectRole_PROJECT_ROLE_OWNER:
			role = "projectOwner"
		case uv1.ProjectRole_PROJECT_ROLE_MEMBER:
			role = "projectMember"
		default:
			return false
		}
	default:
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

type projectAndRoles struct {
	project     *cache.P
	orgRole     uv1.OrganizationRole
	projectRole uv1.ProjectRole
}

func (s *Server) findAssociatedProjectAndRoles(userID, requestedOrgID, requestedProjectID string) (*projectAndRoles, error) {
	// TODO(kenji): When an org ID or a project ID is specified,
	// check if the specified resource belongs to the same tenant as the user.

	userProjects := s.cache.GetProjectsByUserID(userID)
	userOrgs := s.cache.GetOrganizationsByUserID(userID)

	projectID, err := s.findAssociatedProjectID(userID, requestedOrgID, requestedProjectID, userProjects, userOrgs)
	if err != nil {
		return nil, err
	}

	project, ok := s.cache.GetProjectByID(projectID)
	if !ok {
		return nil, fmt.Errorf("project %s not found", requestedProjectID)
	}

	projectRole := uv1.ProjectRole_PROJECT_ROLE_UNSPECIFIED
	for _, p := range userProjects {
		if p.ProjectID == project.ID {
			projectRole = p.Role
			break
		}
	}

	orgRole := uv1.OrganizationRole_ORGANIZATION_ROLE_UNSPECIFIED
	for _, o := range userOrgs {
		if o.OrganizationID == project.OrganizationID {
			orgRole = o.Role
			break
		}
	}
	if orgRole == uv1.OrganizationRole_ORGANIZATION_ROLE_UNSPECIFIED {
		return nil, fmt.Errorf("organization role not found for organizattion %q", project.OrganizationID)
	}

	return &projectAndRoles{
		project:     project,
		orgRole:     orgRole,
		projectRole: projectRole,
	}, nil
}

func (s *Server) findAssociatedProjectID(
	userID,
	requestedOrgID,
	requestedProjectID string,
	userProjects []cache.PU,
	userOrgs []cache.OU,
) (string, error) {
	if requestedProjectID != "" {
		// Use this project. Grab the role if the user belongs to the project and/or the project's organization.

		// TODO(kenji): Check also if the project belongs to the user's tenant.
		p, ok := s.cache.GetProjectByID(requestedProjectID)
		if !ok {
			return "", fmt.Errorf("project %s not found", requestedProjectID)
		}
		// Return an error if the specifies the org ID in the request, but the org ID does not match the org ID
		// of the project.
		if requestedOrgID != "" && requestedOrgID != p.OrganizationID {
			return "", fmt.Errorf("invalid org ID (%q) and project ID (%q) combination", requestedOrgID, requestedProjectID)
		}

		return p.ID, nil
	}

	if requestedOrgID != "" {
		// TODO(kenji): Check also if the org belongs to the user's tenant.
		if _, ok := s.cache.GetOrganizationByID(requestedOrgID); !ok {
			return "", fmt.Errorf("organization %s not found", requestedOrgID)
		}

		// Find the project. First find a project where the user belongs to. If not found,
		// use the first project in the org.

		for _, p := range userProjects {
			if p.OrganizationID == requestedOrgID {
				return p.ProjectID, nil
			}
		}

		// User does not belong to any project. We still need to decide a project for the k8s namespace.
		// Use the first one in the project.
		projects := s.cache.GetProjectsByOrganizationID(requestedOrgID)
		if len(projects) == 0 {
			return "", fmt.Errorf("project not found in the organization %s", requestedOrgID)
		}
		return projects[0].ID, nil
	}

	// When neither org ID nor project ID is specified, first project where the user belongs to and then org.

	if len(userProjects) > 0 {
		return userProjects[0].ProjectID, nil
	}

	for _, o := range userOrgs {
		projects := s.cache.GetProjectsByOrganizationID(o.OrganizationID)
		if len(projects) > 0 {
			return projects[0].ID, nil
		}
	}

	return "", fmt.Errorf("unable to identify a project for the user")
}

func (s *Server) assignedKubernetesEnvs(namespace, tenantID string) []*v1.Project_AssignedKubernetesEnv {
	// TODO(kenji): Revisit. Currently we allow the user to access the project namespace for all registered clusters.
	var envs []*v1.Project_AssignedKubernetesEnv
	for _, c := range s.cache.GetClustersByTenantID(tenantID) {
		envs = append(envs, &v1.Project_AssignedKubernetesEnv{
			ClusterId: c.ID,
			Namespace: namespace,
		})
	}
	return envs
}

func toScope(req *v1.AuthorizeRequest) string {
	return fmt.Sprintf("%s.%s", req.AccessResource, req.Capability)
}
