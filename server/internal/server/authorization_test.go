package server

import (
	"context"
	"testing"

	v1 "github.com/llm-operator/rbac-manager/api/v1"
	"github.com/llm-operator/rbac-manager/server/internal/cache"
	"github.com/llm-operator/rbac-manager/server/internal/dex"
	uv1 "github.com/llm-operator/user-manager/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestAuthorize(t *testing.T) {
	roleScopesMap := map[string][]string{
		"owner": {
			"api.object.read",
			"api.object.write",
		},
	}

	tcs := []struct {
		name     string
		req      *v1.AuthorizeRequest
		apikeys  map[string]*cache.K
		orgroles map[string][]cache.OU
		is       *dex.Introspection
		want     bool
	}{
		{
			name: "authorized with API key",
			req: &v1.AuthorizeRequest{
				Token:          "keySecret",
				AccessResource: "api.object",
				Capability:     "read",
			},
			apikeys: map[string]*cache.K{
				"keySecret": {
					Role: "owner",
				},
			},
			want: true,
		},
		{
			name: "unauthorized with invalid role in API key",
			req: &v1.AuthorizeRequest{
				Token:          "keySecret",
				AccessResource: "api.object",
				Capability:     "read",
			},
			apikeys: map[string]*cache.K{
				"keySecret": {
					Role: "different-role",
				},
			},
			want: false,
		},
		{
			name: "authorized with dex",
			req: &v1.AuthorizeRequest{
				Token:          "jwt",
				AccessResource: "api.object",
				Capability:     "read",
			},
			apikeys: map[string]*cache.K{},
			orgroles: map[string][]cache.OU{
				"my-user": {
					{Role: uv1.OrganizationRole_ORGANIZATION_ROLE_OWNER},
				},
			},
			is: &dex.Introspection{
				Active: true,
				Extra: dex.IntrospectionExtra{
					Email: "my-user",
				},
			},
			want: true,
		},
		{
			name: "unauthorized with inactive token",
			req: &v1.AuthorizeRequest{
				Token:          "jwt",
				AccessResource: "api.object",
				Capability:     "read",
			},
			apikeys: map[string]*cache.K{},
			is: &dex.Introspection{
				Active: false,
				Extra: dex.IntrospectionExtra{
					Email: "my-user",
				},
			},
			want: false,
		},
		{
			name: "unauthorized with invalid user",
			req: &v1.AuthorizeRequest{
				Token:          "jwt",
				AccessResource: "api.object",
				Capability:     "read",
			},
			apikeys: map[string]*cache.K{},
			is: &dex.Introspection{
				Active: true,
				Extra: dex.IntrospectionExtra{
					Email: "different-user",
				},
			},
			want: false,
		},

		{
			name: "unauthorized with scope",
			req: &v1.AuthorizeRequest{
				Token:          "jwt",
				AccessResource: "api.different-object",
				Capability:     "read",
			},
			apikeys: map[string]*cache.K{},
			is: &dex.Introspection{
				Active: true,
				Extra: dex.IntrospectionExtra{
					Email: "my-user",
				},
			},
			want: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			srv := &Server{
				tokenIntrospector: &fakeTokenIntrospector{
					is: tc.is,
				},
				cache: &fakeCacheGetter{
					apikeys:  tc.apikeys,
					orgroles: tc.orgroles,
				},
				roleScopesMapper: roleScopesMap,
			}
			resp, err := srv.Authorize(context.Background(), tc.req)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, resp.Authorized)
		})
	}
}

type fakeTokenIntrospector struct {
	is *dex.Introspection
}

func (f *fakeTokenIntrospector) TokenIntrospect(token string) (*dex.Introspection, error) {
	return f.is, nil
}

type fakeCacheGetter struct {
	apikeys  map[string]*cache.K
	orgroles map[string][]cache.OU
}

func (c *fakeCacheGetter) GetAPIKeyBySecret(secret string) (*cache.K, bool) {
	k, ok := c.apikeys[secret]
	return k, ok
}

func (c *fakeCacheGetter) GetOrganizationsByUserID(userID string) []cache.OU {
	return c.orgroles[userID]
}
