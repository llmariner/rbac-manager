package cache

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	cv1 "github.com/llm-operator/cluster-manager/api/v1"
	"github.com/llm-operator/rbac-manager/server/internal/config"
	uv1 "github.com/llm-operator/user-manager/api/v1"
	"google.golang.org/grpc"
)

// K represents an API key.
type K struct {
	Role           string
	UserID         string
	OrganizationID string
	ProjectID      string
	TenantID       string
}

// C represents a cluster.
type C struct {
	ID       string
	TenantID string
}

// O represents an organization.
type O struct {
	ID       string
	TenantID string
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

// U represents a user.
type U struct {
	ID       string
	TenantID string
}

type userInfoLister interface {
	ListInternalAPIKeys(ctx context.Context, in *uv1.ListInternalAPIKeysRequest, opts ...grpc.CallOption) (*uv1.ListInternalAPIKeysResponse, error)
	ListInternalOrganizations(ctx context.Context, in *uv1.ListInternalOrganizationsRequest, opts ...grpc.CallOption) (*uv1.ListInternalOrganizationsResponse, error)
	ListOrganizationUsers(ctx context.Context, in *uv1.ListOrganizationUsersRequest, opts ...grpc.CallOption) (*uv1.ListOrganizationUsersResponse, error)
	ListProjects(ctx context.Context, in *uv1.ListProjectsRequest, opts ...grpc.CallOption) (*uv1.ListProjectsResponse, error)
	ListProjectUsers(ctx context.Context, in *uv1.ListProjectUsersRequest, opts ...grpc.CallOption) (*uv1.ListProjectUsersResponse, error)
}

type clusterInfoLister interface {
	ListInternalClusters(ctx context.Context, in *cv1.ListInternalClustersRequest, opts ...grpc.CallOption) (*cv1.ListInternalClustersResponse, error)
}

// NewStore creates a new cache store.
func NewStore(
	userInfoLister userInfoLister,
	clusterInfoLister clusterInfoLister,
	debug *config.DebugConfig,
) *Store {
	return &Store{
		userInfoLister:    userInfoLister,
		clusterInfoLister: clusterInfoLister,

		apiKeysBySecret: map[string]*K{},

		clustersByRegistrationKey: map[string]*C{},
		clustersByTenantID:        map[string][]C{},

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
	userInfoLister    userInfoLister
	clusterInfoLister clusterInfoLister

	// apiKeysBySecret is a set of API keys, keyed by its secret.
	apiKeysBySecret map[string]*K

	// clustersByRegistrationKey is a set of clusters, keyed by its registration key.
	clustersByRegistrationKey map[string]*C

	// clustersByTenantID is a set of clusters, keyed by its tenant ID.
	clustersByTenantID map[string][]C

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

	// usersByID is a set of users, keyed by its ID.
	usersByID map[string]*U

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

// GetClusterByRegistrationKey returns a cluster by its registration key.
func (c *Store) GetClusterByRegistrationKey(key string) (*C, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cluster, ok := c.clustersByRegistrationKey[key]
	if !ok {
		return nil, false
	}
	return cluster, true
}

// GetClustersByTenantID returns clusters by its tenant ID.
func (c *Store) GetClustersByTenantID(tenantID string) []C {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.clustersByTenantID[tenantID]
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

// GetUserByID returns a user by its ID.
func (c *Store) GetUserByID(userID string) (*U, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	u, ok := c.usersByID[userID]
	return u, ok
}

// Sync synchronizes the cache.
func (c *Store) Sync(ctx context.Context, interval time.Duration) error {
	if err := c.updateCache(ctx); err != nil {
		// Gracefully ignore the error.
		// TODO(kenji): Make the pod unready.
		log.Printf("Failed to update the cache: %s. Ignoring.", err)
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
	resp, err := c.userInfoLister.ListInternalAPIKeys(ctx, &uv1.ListInternalAPIKeysRequest{})
	if err != nil {
		return err
	}

	m := map[string]*K{}
	for _, apiKey := range resp.ApiKeys {
		m[apiKey.ApiKey.Secret] = &K{
			// TODO(kenji): Fill this properly.
			Role:           c.apiKeyRole,
			UserID:         apiKey.ApiKey.User.Id,
			OrganizationID: apiKey.ApiKey.Organization.Id,
			ProjectID:      apiKey.ApiKey.Project.Id,
			TenantID:       apiKey.TenantId,
		}
	}

	cresp, err := c.clusterInfoLister.ListInternalClusters(ctx, &cv1.ListInternalClustersRequest{})
	if err != nil {
		return err
	}
	cs := map[string]*C{}
	csByTenantID := map[string][]C{}
	for _, cluster := range cresp.Clusters {
		c := C{
			ID:       cluster.Cluster.Id,
			TenantID: cluster.TenantId,
		}
		cs[cluster.Cluster.RegistrationKey] = &c
		csByTenantID[cluster.TenantId] = append(csByTenantID[cluster.TenantId], c)
	}

	orgs, err := c.userInfoLister.ListInternalOrganizations(ctx, &uv1.ListInternalOrganizationsRequest{})
	if err != nil {
		return err
	}
	orgUsers, err := c.userInfoLister.ListOrganizationUsers(ctx, &uv1.ListOrganizationUsersRequest{})
	if err != nil {
		return err
	}
	projects, err := c.userInfoLister.ListProjects(ctx, &uv1.ListProjectsRequest{})
	if err != nil {
		return err
	}
	projectUsers, err := c.userInfoLister.ListProjectUsers(ctx, &uv1.ListProjectUsersRequest{})
	if err != nil {
		return err
	}

	orgsByID := map[string]*O{}
	for _, org := range orgs.Organizations {
		id := org.Organization.Id
		orgsByID[id] = &O{
			ID:       id,
			TenantID: org.TenantId,
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

	usersByID := map[string]*U{}
	for _, user := range orgUsers.Users {
		o, ok := orgsByID[user.OrganizationId]
		if !ok {
			return fmt.Errorf("organization %s not found for user %s", user.OrganizationId, user.UserId)
		}

		if existing, ok := usersByID[user.UserId]; ok {
			if existing.TenantID != o.TenantID {
				return fmt.Errorf("user %s has multiple tenant IDs", user.UserId)
			}
			continue
		}

		usersByID[user.UserId] = &U{
			ID:       user.UserId,
			TenantID: o.TenantID,
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.apiKeysBySecret = m

	c.clustersByRegistrationKey = cs
	c.clustersByTenantID = csByTenantID

	c.orgsByID = orgsByID
	c.orgsByUserID = orgsByUserID

	c.projectsByID = projectsByID
	c.projectsByOrganizationID = projectsByOrganizationID
	c.projectsByUserID = projectsByUserID

	c.usersByID = usersByID

	return nil
}
