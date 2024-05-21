package cache

import (
	"context"
	"fmt"
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
}

// O represents a role associated with a organization user.
type O struct {
	Role                uv1.OrganizationRole
	OrganizationID      string
	KubernetesNamespace string
}

type userInfoLister interface {
	ListAPIKeys(ctx context.Context, in *uv1.ListAPIKeysRequest, opts ...grpc.CallOption) (*uv1.ListAPIKeysResponse, error)
	ListOrganizations(ctx context.Context, in *uv1.ListOrganizationsRequest, opts ...grpc.CallOption) (*uv1.ListOrganizationsResponse, error)
	ListOrganizationUsers(ctx context.Context, in *uv1.ListOrganizationUsersRequest, opts ...grpc.CallOption) (*uv1.ListOrganizationUsersResponse, error)
}

// NewStore creates a new cache store.
func NewStore(
	userInfoLister userInfoLister,
	debug *config.DebugConfig,
) *Store {
	return &Store{
		userInfoLister:  userInfoLister,
		apiKeysBySecret: map[string]*K{},
		orgsByUserID:    map[string][]O{},
		apiKeyRole:      debug.APIKeyRole,
	}
}

// Store is a cache for API keys and organization users.
type Store struct {
	userInfoLister userInfoLister

	// apiKeysBySecret is a set of API keys, keyed by its secret.
	apiKeysBySecret map[string]*K
	// orgsByUserID is a set of organization users, keyed by its user ID.
	orgsByUserID map[string][]O
	mu           sync.RWMutex

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

// GetOrganizationsByUserID returns organization users by its user ID.
func (c *Store) GetOrganizationsByUserID(userID string) ([]O, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	users, ok := c.orgsByUserID[userID]
	if !ok {
		return nil, false
	}
	return users, true
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
	resp, err := c.userInfoLister.ListAPIKeys(ctx, &uv1.ListAPIKeysRequest{})
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
		}
	}

	orgs, err := c.userInfoLister.ListOrganizations(ctx, &uv1.ListOrganizationsRequest{})
	if err != nil {
		return err
	}
	orgsByID := map[string]*uv1.Organization{}
	for _, org := range orgs.Organizations {
		orgsByID[org.Id] = org
	}

	orgUsers, err := c.userInfoLister.ListOrganizationUsers(ctx, &uv1.ListOrganizationUsersRequest{})
	if err != nil {
		return err
	}
	orguserByUserID := map[string][]O{}
	for _, user := range orgUsers.Users {
		o, ok := orgsByID[user.OrganizationId]
		if !ok {
			return fmt.Errorf("Organization not found: %s", user.OrganizationId)
		}
		orguserByUserID[user.UserId] = append(orguserByUserID[user.UserId], O{
			OrganizationID:      user.OrganizationId,
			Role:                user.Role,
			KubernetesNamespace: o.KubernetesNamespace,
		})
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.apiKeysBySecret = m
	c.orgsByUserID = orguserByUserID
	return nil
}
