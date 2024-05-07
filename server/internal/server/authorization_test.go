package server

import (
	"context"
	"testing"

	v1 "github.com/llm-operator/rbac-manager/api/v1"
	"github.com/llm-operator/rbac-manager/server/internal/apikey"
	"github.com/llm-operator/rbac-manager/server/internal/config"
	"github.com/llm-operator/rbac-manager/server/internal/dex"
	"github.com/stretchr/testify/assert"
)

func TestAuthorize(t *testing.T) {
	debug := &config.DebugConfig{
		UserOrgMap: map[string]string{
			"my-user": "my-org",
		},
		OrgRoleMap: map[string]string{
			"my-org": "all",
		},
		RoleScopesMap: map[string][]string{
			"all": {
				"api.object.read",
				"api.object.write",
			},
		},
	}

	tcs := []struct {
		name  string
		req   *v1.AuthorizeRequest
		cache map[string]*apikey.K
		is    *dex.Introspection
		want  bool
	}{
		{
			name: "authorized with API key",
			req: &v1.AuthorizeRequest{
				Token: "keyID",
				Scope: "api.object.read",
			},
			cache: map[string]*apikey.K{
				"keyID": {
					Role: "all",
				},
			},
			want: true,
		},
		{
			name: "unauthorized with invalid role in API key",
			req: &v1.AuthorizeRequest{
				Token: "keyID",
				Scope: "api.object.read",
			},
			cache: map[string]*apikey.K{
				"keyID": {
					Role: "different-role",
				},
			},
			want: false,
		},
		{
			name: "authorized with dex",
			req: &v1.AuthorizeRequest{
				Token: "keyID",
				Scope: "api.object.read",
			},
			cache: map[string]*apikey.K{},
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
				Token: "keyID",
				Scope: "api.object.read",
			},
			cache: map[string]*apikey.K{},
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
				Token: "keyID",
				Scope: "api.object.read",
			},
			cache: map[string]*apikey.K{},
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
				Token: "keyID",
				Scope: "api.different-object.read",
			},
			cache: map[string]*apikey.K{},
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
				apiKeyCache: &fakeAPIKeyCache{
					cache: tc.cache,
				},

				userOrgMapper:    debug.UserOrgMap,
				orgRoleMapper:    debug.OrgRoleMap,
				roleScopesMapper: debug.RoleScopesMap,
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

type fakeAPIKeyCache struct {
	cache map[string]*apikey.K
}

func (c *fakeAPIKeyCache) GetAPIKey(keyID string) (*apikey.K, bool) {
	k, ok := c.cache[keyID]
	return k, ok
}
