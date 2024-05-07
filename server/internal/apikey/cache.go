package apikey

import (
	"context"
	"sync"
	"time"

	"github.com/llm-operator/rbac-manager/server/internal/config"
	uv1 "github.com/llm-operator/user-manager/api/v1"
	"google.golang.org/grpc"
)

// K represents a role associated with an API key.
type K struct {
	Role string
}

type apiKeyLister interface {
	ListAPIKeys(ctx context.Context, in *uv1.ListAPIKeysRequest, opts ...grpc.CallOption) (*uv1.ListAPIKeysResponse, error)
}

// NewCache creates a new cache.
func NewCache(
	apiKeyLister apiKeyLister,
	debug *config.DebugConfig,
) *Cache {
	return &Cache{
		apiKeyLister:    apiKeyLister,
		apiKeysBySecret: map[string]*K{},
		apiKeyRole:      debug.APIKeyRole,
	}
}

// Cache is a cache for API keys.
type Cache struct {
	apiKeyLister apiKeyLister

	// apiKeysBySecret is a set of API keys, keyed by its secret.
	apiKeysBySecret map[string]*K
	mu              sync.RWMutex

	apiKeyRole string
}

// GetAPIKeyBySecret returns an API key by its secret.
func (c *Cache) GetAPIKeyBySecret(secret string) (*K, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	k, ok := c.apiKeysBySecret[secret]
	if !ok {
		return nil, false
	}
	return k, true
}

// Sync synchronizes the cache.
func (c *Cache) Sync(ctx context.Context, interval time.Duration) error {
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
				return err
			}
		}
	}
}

func (c *Cache) updateCache(ctx context.Context) error {
	resp, err := c.apiKeyLister.ListAPIKeys(ctx, &uv1.ListAPIKeysRequest{})
	if err != nil {
		return err
	}

	m := map[string]*K{}
	for _, apiKey := range resp.Data {
		m[apiKey.Secret] = &K{
			// TODO(kenji): Fill this properly.
			Role: c.apiKeyRole,
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.apiKeysBySecret = m
	return nil
}
