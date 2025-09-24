package server

import (
	"context"
	"testing"

	v1 "github.com/llmariner/rbac-manager/api/v1"
	"github.com/llmariner/rbac-manager/server/internal/cache"
	"github.com/llmariner/rbac-manager/server/internal/token"
	uv1 "github.com/llmariner/user-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
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
		usersByID                map[string]*cache.U
		is                       *token.Introspection
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
					ProjectID:                "my-project",
					OrganizationID:           "my-org",
					OrganizationRole:         uv1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
					ProjectRole:              uv1.ProjectRole_PROJECT_ROLE_OWNER,
					ExcludedFromRateLimiting: false,
				},
			},
			orgsByID: map[string]*cache.O{
				"my-org": {
					ID: "my-org",
				},
			},
			projectsByID: map[string]*cache.P{
				"my-project": {
					KubernetesNamespace: "ns",
					OrganizationID:      "my-org",
				},
			},
			usersByID: map[string]*cache.U{},
			want:      true,
		},
		{
			name: "authorized with API key excluded from rate limiting",
			req: &v1.AuthorizeRequest{
				Token:          "keySecretExcluded",
				AccessResource: "api.object",
				Capability:     "read",
			},
			apikeys: map[string]*cache.K{
				"keySecretExcluded": {
					ProjectID:                "my-project",
					OrganizationID:           "my-org",
					OrganizationRole:         uv1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
					ProjectRole:              uv1.ProjectRole_PROJECT_ROLE_OWNER,
					ExcludedFromRateLimiting: true,
				},
			},
			orgsByID: map[string]*cache.O{
				"my-org": {
					ID: "my-org",
				},
			},
			projectsByID: map[string]*cache.P{
				"my-project": {
					KubernetesNamespace: "ns",
					OrganizationID:      "my-org",
				},
			},
			usersByID: map[string]*cache.U{},
			want:      true,
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
					OrganizationRole: uv1.OrganizationRole_ORGANIZATION_ROLE_UNSPECIFIED,
					ProjectRole:      uv1.ProjectRole_PROJECT_ROLE_UNSPECIFIED,
				},
			},
			orgsByID: map[string]*cache.O{
				"my-org": {
					ID: "my-org",
				},
			},
			projectsByID: map[string]*cache.P{
				"my-project": {
					KubernetesNamespace: "ns",
					OrganizationID:      "my-org",
				},
			},
			usersByID: map[string]*cache.U{},
			want:      false,
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
			usersByID: map[string]*cache.U{
				"my-user": {
					ID:       "my-user",
					TenantID: "t0",
				},
			},
			is: &token.Introspection{
				Active: true,
				Extra: token.IntrospectionExtra{
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
			is: &token.Introspection{
				Active: false,
				Extra: token.IntrospectionExtra{
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
			is: &token.Introspection{
				Active: true,
				Extra: token.IntrospectionExtra{
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
			is: &token.Introspection{
				Active: true,
				Extra: token.IntrospectionExtra{
					Email: "my-user",
				},
			},
			want: false,
		},
		{
			name: "authorized with no-user",
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
			is: &token.Introspection{
				Active: true,
				Extra: token.IntrospectionExtra{
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
					usersByID:                tc.usersByID,
				},
				roleScopesMapper: roleScopesMap,
			}
			resp, err := srv.Authorize(context.Background(), tc.req)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, resp.Authorized)
			excludedFromRateLimiting := false
			if tc.apikeys != nil {
				if k, ok := tc.apikeys[tc.req.Token]; ok {
					excludedFromRateLimiting = k.ExcludedFromRateLimiting
				}
			}
			assert.Equal(t, excludedFromRateLimiting, resp.ExcludedFromRateLimiting)
		})
	}
}

func TestFindAssociatedProjectAndRoles(t *testing.T) {
	const userID = "u0"
	org0 := cache.O{
		ID: "o0",
	}
	org1 := cache.O{
		ID: "o1",
	}
	project0 := cache.P{
		ID:                  "p0",
		OrganizationID:      org0.ID,
		KubernetesNamespace: "n0",
	}
	project1 := cache.P{
		ID:                  "p1",
		OrganizationID:      org0.ID,
		KubernetesNamespace: "n1",
	}
	project2 := cache.P{
		ID:                  "p2",
		OrganizationID:      org1.ID,
		KubernetesNamespace: "n2",
	}

	cache := &fakeCacheGetter{
		orgsByID: map[string]*cache.O{
			org0.ID: &org0,
			org1.ID: &org1,
		},
		orgsByUserID: map[string][]cache.OU{
			userID: {
				{
					Role:           uv1.OrganizationRole_ORGANIZATION_ROLE_READER,
					OrganizationID: org0.ID,
				},
				{
					Role:           uv1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
					OrganizationID: org1.ID,
				},
			},
		},
		projectsByID: map[string]*cache.P{
			project0.ID: &project0,
			project1.ID: &project1,
			project2.ID: &project2,
		},
		projectsByOrganizationID: map[string][]cache.P{
			org0.ID: {project0, project1},
			org1.ID: {project2},
		},
		projectsByUserID: map[string][]cache.PU{
			userID: {
				{
					Project: &project0,
					Role:    uv1.ProjectRole_PROJECT_ROLE_OWNER,
				},
				{
					Project: &project1,
					Role:    uv1.ProjectRole_PROJECT_ROLE_MEMBER,
				},
			},
		},
		usersByID: map[string]*cache.U{
			userID: {
				ID:       userID,
				TenantID: "t0",
			},
		},
	}

	tcs := []struct {
		name               string
		requestedOrgID     string
		requestedProjectID string
		want               *projectAndRoles
		wantErr            bool
	}{
		{
			name:               "requested project id p0",
			requestedOrgID:     "",
			requestedProjectID: project0.ID,
			want: &projectAndRoles{
				project:     &project0,
				orgRole:     uv1.OrganizationRole_ORGANIZATION_ROLE_READER,
				projectRole: uv1.ProjectRole_PROJECT_ROLE_OWNER,
			},
		},
		{
			name:               "requested project id p1",
			requestedOrgID:     "",
			requestedProjectID: project1.ID,
			want: &projectAndRoles{
				project:     &project1,
				orgRole:     uv1.OrganizationRole_ORGANIZATION_ROLE_READER,
				projectRole: uv1.ProjectRole_PROJECT_ROLE_MEMBER,
			},
		},
		{
			name:               "requested project id p2",
			requestedOrgID:     "",
			requestedProjectID: project2.ID,
			want: &projectAndRoles{
				project:     &project2,
				orgRole:     uv1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
				projectRole: uv1.ProjectRole_PROJECT_ROLE_UNSPECIFIED,
			},
		},
		{
			name:               "uknown requested project id",
			requestedOrgID:     "",
			requestedProjectID: "unknown",
			wantErr:            true,
		},
		{
			name:               "requested org id o0",
			requestedOrgID:     org0.ID,
			requestedProjectID: "",
			want: &projectAndRoles{
				project:     &project0,
				orgRole:     uv1.OrganizationRole_ORGANIZATION_ROLE_READER,
				projectRole: uv1.ProjectRole_PROJECT_ROLE_OWNER,
			},
		},
		{
			name:               "requested org id o1",
			requestedOrgID:     org1.ID,
			requestedProjectID: "",
			want: &projectAndRoles{
				project:     &project2,
				orgRole:     uv1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
				projectRole: uv1.ProjectRole_PROJECT_ROLE_UNSPECIFIED,
			},
		},
		{
			name:               "requested org id and project id",
			requestedOrgID:     org0.ID,
			requestedProjectID: project0.ID,
			want: &projectAndRoles{
				project:     &project0,
				orgRole:     uv1.OrganizationRole_ORGANIZATION_ROLE_READER,
				projectRole: uv1.ProjectRole_PROJECT_ROLE_OWNER,
			},
		},
		{
			name:               "mismatching requested org id and project id",
			requestedOrgID:     org0.ID,
			requestedProjectID: project2.ID,
			wantErr:            true,
		},
		{
			name:               "no project id and org id",
			requestedOrgID:     "",
			requestedProjectID: "",
			want: &projectAndRoles{
				project:     &project0,
				orgRole:     uv1.OrganizationRole_ORGANIZATION_ROLE_READER,
				projectRole: uv1.ProjectRole_PROJECT_ROLE_OWNER,
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			srv := &Server{
				cache: cache,
			}
			resp, err := srv.findAssociatedProjectAndRoles(userID, tc.requestedOrgID, tc.requestedProjectID)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, *tc.want.project, *resp.project)
			assert.Equal(t, tc.want.orgRole, resp.orgRole)
			assert.Equal(t, tc.want.projectRole, resp.projectRole)
		})
	}
}

func TestAssignedKubernetesEnvsInternal(t *testing.T) {
	clusters := []cache.C{
		{
			ID: "c0",
		},
		{
			ID: "c1",
		},
	}
	tcs := []struct {
		name        string
		namespace   string
		assignments []*uv1.ProjectAssignment
		want        []*v1.Project_AssignedKubernetesEnv
	}{
		{
			name:      "only kubernetes namespace",
			namespace: "ns0",
			want: []*v1.Project_AssignedKubernetesEnv{
				{
					ClusterId: "c0",
					Namespace: "ns0",
				},
				{
					ClusterId: "c1",
					Namespace: "ns0",
				},
			},
		},
		{
			name: "only assignments",
			assignments: []*uv1.ProjectAssignment{
				{
					ClusterId: "",
					Namespace: "ns0",
				},
				{
					ClusterId: "c1",
					Namespace: "ns1",
				},
			},
			want: []*v1.Project_AssignedKubernetesEnv{
				{
					ClusterId: "c0",
					Namespace: "ns0",
				},
				{
					ClusterId: "c1",
					Namespace: "ns1",
				},
				{
					ClusterId: "c1",
					Namespace: "ns0",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := assignedKubernetesEnvsInternal(tc.namespace, tc.assignments, clusters)
			assert.Len(t, got, len(tc.want))
			for i, g := range got {
				w := tc.want[i]
				assert.Truef(t, proto.Equal(w, g), "wanted %+v, but got %+v", w, g)
			}
		})
	}
}

type fakeTokenIntrospector struct {
	is *token.Introspection
}

func (f *fakeTokenIntrospector) TokenIntrospect(token string) (*token.Introspection, error) {
	return f.is, nil
}

type fakeCacheGetter struct {
	apikeys map[string]*cache.K

	clusters map[string]*cache.C

	orgsByID     map[string]*cache.O
	orgsByUserID map[string][]cache.OU

	projectsByID             map[string]*cache.P
	projectsByOrganizationID map[string][]cache.P
	projectsByUserID         map[string][]cache.PU

	usersByID map[string]*cache.U
}

func (c *fakeCacheGetter) GetAPIKeyBySecret(secret string) (*cache.K, bool) {
	k, ok := c.apikeys[secret]
	return k, ok
}

func (c *fakeCacheGetter) GetClusterByRegistrationKey(key string) (*cache.C, bool) {
	cl, ok := c.clusters[key]
	return cl, ok
}

func (c *fakeCacheGetter) GetClustersByTenantID(tenantID string) []cache.C {
	var clusters []cache.C
	for _, cl := range c.clusters {
		if cl.TenantID == tenantID {
			clusters = append(clusters, *cl)
		}
	}
	return clusters
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

func (c *fakeCacheGetter) GetUserByID(id string) (*cache.U, bool) {
	u, ok := c.usersByID[id]
	return u, ok
}
