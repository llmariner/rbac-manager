package server

import (
	"context"
	"testing"

	v1 "github.com/llm-operator/rbac-manager/api/v1"
	"github.com/llm-operator/rbac-manager/server/internal/cache"
	"github.com/llm-operator/rbac-manager/server/internal/dex"
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
		orgroles map[string][]cache.O
		is       *dex.Introspection
		want     bool
	}{
		{
			name: "authorized with API key",
			req: &v1.AuthorizeRequest{
				Token: "keySecret",
				Scope: "api.object.read",
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
				Token: "keySecret",
				Scope: "api.object.read",
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
				Token: "jwt",
				Scope: "api.object.read",
			},
			apikeys: map[string]*cache.K{},
			orgroles: map[string][]cache.O{
				"my-user": {
					{Role: "owner"},
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
				Token: "jwt",
				Scope: "api.object.read",
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
				Token: "jwt",
				Scope: "api.object.read",
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
				Token: "jwt",
				Scope: "api.different-object.read",
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
	orgroles map[string][]cache.O
}

func (c *fakeCacheGetter) GetAPIKeyBySecret(secret string) (*cache.K, bool) {
	k, ok := c.apikeys[secret]
	return k, ok
}

func (c *fakeCacheGetter) GetOrganizationsByUserID(userID string) ([]cache.O, bool) {
	users, ok := c.orgroles[userID]
	return users, ok
}
