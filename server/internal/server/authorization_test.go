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
		"organizationOwner": {
			"api.object.read",
			"api.object.write",
		},
		"projectOwner": {
			"api.object.read",
			"api.object.write",
		},
	}

	tcs := []struct {
		name                     string
		req                      *v1.AuthorizeRequest
		apikeys                  map[string]*cache.K
		orgsByID                 map[string]*cache.O
		orgsByUserID             map[string][]cache.OU
		projectsByID             map[string]*cache.P
		projectsByOrganizationID map[string][]cache.P
		is                       *dex.Introspection
		want                     bool
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
					ProjectID: "my-project",
					Role:      "projectOwner",
				},
			},
			projectsByID: map[string]*cache.P{
				"my-project": {
					KubernetesNamespace: "ns",
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
			projectsByID: map[string]*cache.P{
				"my-project": {
					KubernetesNamespace: "ns",
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
			orgsByID: map[string]*cache.O{
				"my-org": {
					ID: "my-org",
				},
			},
			orgsByUserID: map[string][]cache.OU{
				"my-user": {
					{
						Role:           uv1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
						OrganizationID: "my-org",
					},
				},
			},
			projectsByID: map[string]*cache.P{
				"my-project": {
					ID:                  "my-project",
					OrganizationID:      "my-org",
					KubernetesNamespace: "ns",
				},
			},
			projectsByOrganizationID: map[string][]cache.P{
				"my-org": {
					{
						ID:                  "my-project",
						OrganizationID:      "my-org",
						KubernetesNamespace: "ns",
					},
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
					apikeys:                  tc.apikeys,
					orgsByID:                 tc.orgsByID,
					orgsByUserID:             tc.orgsByUserID,
					projectsByID:             tc.projectsByID,
					projectsByOrganizationID: tc.projectsByOrganizationID,
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
	apikeys map[string]*cache.K

	orgsByID     map[string]*cache.O
	orgsByUserID map[string][]cache.OU

	projectsByID             map[string]*cache.P
	projectsByOrganizationID map[string][]cache.P
	projectsByUserID         map[string][]cache.PU
}

func (c *fakeCacheGetter) GetAPIKeyBySecret(secret string) (*cache.K, bool) {
	k, ok := c.apikeys[secret]
	return k, ok
}

func (c *fakeCacheGetter) GetOrganizationByID(organizationID string) (*cache.O, bool) {
	o, ok := c.orgsByID[organizationID]
	return o, ok
}

func (c *fakeCacheGetter) GetOrganizationsByUserID(userID string) []cache.OU {
	return c.orgsByUserID[userID]
}

func (c *fakeCacheGetter) GetProjectsByOrganizationID(organizationID string) []cache.P {
	return c.projectsByOrganizationID[organizationID]
}

func (c *fakeCacheGetter) GetProjectByID(projectID string) (*cache.P, bool) {
	p, ok := c.projectsByID[projectID]
	return p, ok
}

func (c *fakeCacheGetter) GetProjectsByUserID(userID string) []cache.PU {
	return c.projectsByUserID[userID]
}
