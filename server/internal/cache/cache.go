package cache

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/llm-operator/rbac-manager/server/internal/config"
	uv1 "github.com/llm-operator/user-manager/api/v1"
	"google.golang.org/grpc"
)

// K represents a role associated with an API key.
type K struct {
	Role           string
	UserID         string
	OrganizationID string
	ProjectID      string
}

// O represents an organization.
type O struct {
	ID string
}

// OU represents a role associated with a organization user.
type OU struct {
	Role           uv1.OrganizationRole
	OrganizationID string
}

// P represents a project.
type P struct {
	ID                  string
	OrganizationID      string
	KubernetesNamespace string
}

// PU represents a role associated with a project user.
type PU struct {
	Role           uv1.ProjectRole
	ProjectID      string
	OrganizationID string
}

type userInfoListerFactory interface {
	Create() (UserInfoLister, error)
}

// UserInfoLister is an interface for listing user information.
type UserInfoLister interface {
	ListAPIKeys(ctx context.Context, in *uv1.ListAPIKeysRequest, opts ...grpc.CallOption) (*uv1.ListAPIKeysResponse, error)
	ListOrganizations(ctx context.Context, in *uv1.ListOrganizationsRequest, opts ...grpc.CallOption) (*uv1.ListOrganizationsResponse, error)
	ListOrganizationUsers(ctx context.Context, in *uv1.ListOrganizationUsersRequest, opts ...grpc.CallOption) (*uv1.ListOrganizationUsersResponse, error)
	ListProjects(ctx context.Context, in *uv1.ListProjectsRequest, opts ...grpc.CallOption) (*uv1.ListProjectsResponse, error)
	ListProjectUsers(ctx context.Context, in *uv1.ListProjectUsersRequest, opts ...grpc.CallOption) (*uv1.ListProjectUsersResponse, error)
}

// NewStore creates a new cache store.
func NewStore(
	userInfoListerFactory userInfoListerFactory,
	debug *config.DebugConfig,
) *Store {
	return &Store{
		userInfoListerFactory: userInfoListerFactory,
		apiKeysBySecret:       map[string]*K{},

		orgsByID:     map[string]*O{},
		orgsByUserID: map[string][]OU{},

		projectsByID:             map[string]*P{},
		projectsByOrganizationID: map[string][]P{},
		projectsByUserID:         map[string][]PU{},

		apiKeyRole: debug.APIKeyRole,
	}
}

// Store is a cache for API keys and organization users.
type Store struct {
	userInfoListerFactory userInfoListerFactory

	// apiKeysBySecret is a set of API keys, keyed by its secret.
	apiKeysBySecret map[string]*K

	// orgsByID is a set of organizations, keyed by its ID.
	orgsByID map[string]*O
	// orgsByUserID is a set of organization users, keyed by its user ID.
	orgsByUserID map[string][]OU

	// projectsByID is a set of projects, keyed by its ID.
	projectsByID map[string]*P
	// projectsByOrganizationID is a set of project users, keyed by its organization ID.
	projectsByOrganizationID map[string][]P
	// projectsByUserID is a set of project users, keyed by its user ID.
	projectsByUserID map[string][]PU

	mu sync.RWMutex

	apiKeyRole string
}

// GetAPIKeyBySecret returns an API key by its secret.
func (c *Store) GetAPIKeyBySecret(secret string) (*K, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	k, ok := c.apiKeysBySecret[secret]
	if !ok {
		return nil, false
	}
	return k, true
}

// GetOrganizationByID returns an organization by its ID.
func (c *Store) GetOrganizationByID(organizationID string) (*O, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	o, ok := c.orgsByID[organizationID]
	return o, ok
}

// GetOrganizationsByUserID returns organization users by its user ID.
func (c *Store) GetOrganizationsByUserID(userID string) []OU {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.orgsByUserID[userID]
}

// GetProjectsByOrganizationID returns projects by its organization ID.
func (c *Store) GetProjectsByOrganizationID(organizationID string) []P {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.projectsByOrganizationID[organizationID]
}

// GetProjectByID returns a project by its ID.
func (c *Store) GetProjectByID(projectID string) (*P, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.projectsByID[projectID]
	return p, ok
}

// GetProjectsByUserID returns project users by its user ID.
func (c *Store) GetProjectsByUserID(userID string) []PU {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.projectsByUserID[userID]
}

// Sync synchronizes the cache.
func (c *Store) Sync(ctx context.Context, interval time.Duration) error {
	if err := c.updateCache(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.updateCache(ctx); err != nil {
				// Gracefully ignore the error.
				// TODO(kenji): Make the pod unready.
				log.Printf("Failed to update the cache: %s. Ignoring.", err)
			}
		}
	}
}

func (c *Store) updateCache(ctx context.Context) error {
	lister, err := c.userInfoListerFactory.Create()
	if err != nil {
		return err
	}

	resp, err := lister.ListAPIKeys(ctx, &uv1.ListAPIKeysRequest{})
	if err != nil {
		return err
	}

	m := map[string]*K{}
	for _, apiKey := range resp.Data {
		m[apiKey.Secret] = &K{
			// TODO(kenji): Fill this properly.
			Role:           c.apiKeyRole,
			UserID:         apiKey.User.Id,
			OrganizationID: apiKey.Organization.Id,
			ProjectID:      apiKey.Project.Id,
		}
	}

	orgs, err := lister.ListOrganizations(ctx, &uv1.ListOrganizationsRequest{})
	if err != nil {
		return err
	}
	orgUsers, err := lister.ListOrganizationUsers(ctx, &uv1.ListOrganizationUsersRequest{})
	if err != nil {
		return err
	}
	projects, err := lister.ListProjects(ctx, &uv1.ListProjectsRequest{})
	if err != nil {
		return err
	}
	projectUsers, err := lister.ListProjectUsers(ctx, &uv1.ListProjectUsersRequest{})
	if err != nil {
		return err
	}

	orgsByID := map[string]*O{}
	for _, org := range orgs.Organizations {
		orgsByID[org.Id] = &O{
			ID: org.Id,
		}
	}

	orgsByUserID := map[string][]OU{}
	for _, user := range orgUsers.Users {
		orgsByUserID[user.UserId] = append(orgsByUserID[user.UserId], OU{
			OrganizationID: user.OrganizationId,
			Role:           user.Role,
		})
	}

	projectsByID := map[string]*P{}
	projectsByOrganizationID := map[string][]P{}
	for _, p := range projects.Projects {
		oid := p.OrganizationId
		val := P{
			ID:                  p.Id,
			OrganizationID:      oid,
			KubernetesNamespace: p.KubernetesNamespace,
		}
		projectsByID[p.Id] = &val
		projectsByOrganizationID[oid] = append(projectsByOrganizationID[oid], val)
	}

	projectsByUserID := map[string][]PU{}
	for _, user := range projectUsers.Users {
		projectsByUserID[user.UserId] = append(projectsByUserID[user.UserId], PU{
			ProjectID:      user.ProjectId,
			OrganizationID: user.OrganizationId,
			Role:           user.Role,
		})
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.apiKeysBySecret = m

	c.orgsByID = orgsByID
	c.orgsByUserID = orgsByUserID

	c.projectsByID = projectsByID
	c.projectsByOrganizationID = projectsByOrganizationID
	c.projectsByUserID = projectsByUserID

	return nil
}
