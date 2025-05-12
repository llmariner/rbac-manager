package cache

import (
	"context"
	"testing"

	cv1 "github.com/llmariner/cluster-manager/api/v1"
	uv1 "github.com/llmariner/user-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestCache(t *testing.T) {
	ul := &fakeUserInfoLister{
		apikeys: &uv1.ListInternalAPIKeysResponse{
			ApiKeys: []*uv1.InternalAPIKey{
				{
					ApiKey: &uv1.APIKey{
						Id:           "id0",
						Secret:       "s0",
						User:         &uv1.User{Id: "u0", InternalId: "iu0"},
						Organization: &uv1.Organization{Id: "o0"},
						Project:      &uv1.Project{Id: "p0"},

						OrganizationRole: uv1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
						ProjectRole:      uv1.ProjectRole_PROJECT_ROLE_OWNER,
					},
					TenantId: "tid0",
				},
				{
					ApiKey: &uv1.APIKey{
						Id:           "id1",
						Secret:       "s1",
						User:         &uv1.User{Id: "u0", InternalId: "iu1"},
						Organization: &uv1.Organization{Id: "o1"},
						Project:      &uv1.Project{Id: "p1"},

						OrganizationRole: uv1.OrganizationRole_ORGANIZATION_ROLE_READER,
						ProjectRole:      uv1.ProjectRole_PROJECT_ROLE_MEMBER,
					},
					TenantId: "tid1",
				},
			},
		},
		orgs: &uv1.ListInternalOrganizationsResponse{
			Organizations: []*uv1.InternalOrganization{
				{
					Organization: &uv1.Organization{
						Id: "o0",
					},
					TenantId: "tid0",
				},
				{
					Organization: &uv1.Organization{
						Id: "o1",
					},
					TenantId: "tid0",
				},
			},
		},
		orgusers: &uv1.ListOrganizationUsersResponse{
			Users: []*uv1.OrganizationUser{
				{
					UserId:         "u0",
					InternalUserId: "iu0",
					OrganizationId: "o0",
					Role:           uv1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
				},
				{
					UserId:         "u0",
					InternalUserId: "iu0",
					OrganizationId: "o1",
					Role:           uv1.OrganizationRole_ORGANIZATION_ROLE_READER,
				},
			},
		},
		projects: &uv1.ListProjectsResponse{
			Projects: []*uv1.Project{
				{
					Id:                  "p0",
					OrganizationId:      "o0",
					KubernetesNamespace: "ns0",
				},
				{
					Id:                  "p1",
					OrganizationId:      "o1",
					KubernetesNamespace: "ns1",
				},
			},
		},
		projectusers: &uv1.ListProjectUsersResponse{
			Users: []*uv1.ProjectUser{
				{
					UserId:    "u0",
					ProjectId: "p0",
					Role:      uv1.ProjectRole_PROJECT_ROLE_OWNER,
				},
				{
					UserId:    "u0",
					ProjectId: "p1",
					Role:      uv1.ProjectRole_PROJECT_ROLE_MEMBER,
				},
			},
		},
	}

	cl := &fakeClusterInfoLister{
		clusters: &cv1.ListInternalClustersResponse{
			Clusters: []*cv1.InternalCluster{
				{
					Cluster: &cv1.Cluster{
						Id:              "cid0",
						RegistrationKey: "rkey0",
					},
					TenantId: "tid0",
				},
				{
					Cluster: &cv1.Cluster{
						Id:              "cid1",
						RegistrationKey: "rkey1",
					},
					TenantId: "tid0",
				},
			},
		},
	}

	c := NewStore(ul, cl)
	ctx := context.Background()
	go func() {
		err := c.updateCache(ctx)
		assert.NoError(t, err)
	}()

	err := c.WaitForSync(ctx)
	assert.NoError(t, err)

	wantKeys := map[string]*K{
		"s0": {
			KeyID:          "id0",
			UserID:         "u0",
			InternalUserID: "iu0",
			OrganizationID: "o0",

			OrganizationRole: uv1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
			ProjectRole:      uv1.ProjectRole_PROJECT_ROLE_OWNER,
		},
		"s1": {
			KeyID:          "id1",
			UserID:         "u1",
			InternalUserID: "iu1",
			OrganizationID: "o1",

			OrganizationRole: uv1.OrganizationRole_ORGANIZATION_ROLE_READER,
			ProjectRole:      uv1.ProjectRole_PROJECT_ROLE_MEMBER,
		},
		"s2": nil,
	}

	for k, v := range wantKeys {
		got, ok := c.GetAPIKeyBySecret(k)
		if v == nil {
			assert.False(t, ok)
			continue
		}

		assert.True(t, ok)
		assert.NotNil(t, got)
		assert.Equal(t, v.KeyID, got.KeyID)
		assert.Equal(t, v.OrganizationRole, got.OrganizationRole)
		assert.Equal(t, v.ProjectRole, got.ProjectRole)
	}

	wantClusters := map[string]*C{
		"rkey0": {
			ID: "cid0",
		},
		"rkey1": {
			ID: "cid1",
		},
	}

	for k, v := range wantClusters {
		got, ok := c.GetClusterByRegistrationKey(k)
		assert.True(t, ok)
		assert.Equal(t, v.ID, got.ID)
	}
	wantClustersByTenantID := map[string][]C{
		"tid0": {
			{
				ID:       "cid0",
				TenantID: "tid0",
			},
			{
				ID:       "cid1",
				TenantID: "tid0",
			},
		},
	}
	for k, want := range wantClustersByTenantID {
		got := c.GetClustersByTenantID(k)
		assert.ElementsMatch(t, want, got)
	}

	wantOrgs := map[string]*O{
		"o0": {
			ID:       "o0",
			TenantID: "tid0",
		},
		"o1": {
			ID:       "o1",
			TenantID: "tid0",
		},
	}
	for id, want := range wantOrgs {
		got, ok := c.GetOrganizationByID(id)
		assert.True(t, ok)
		assert.Equal(t, want, got)
	}

	userorgs := c.GetOrganizationsByUserID("u0")
	assert.Len(t, userorgs, 2)
	userorgsByOrg := map[string]*OU{}
	for _, uo := range userorgs {
		userorgsByOrg[uo.OrganizationID] = &uo
	}
	wantOUs := map[string]*OU{
		"o0": {
			Role:           uv1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
			OrganizationID: "o0",
		},
		"o1": {
			Role:           uv1.OrganizationRole_ORGANIZATION_ROLE_READER,
			OrganizationID: "o1",
		},
	}
	for orgID, want := range wantOUs {
		got, ok := userorgsByOrg[orgID]
		assert.True(t, ok)
		assert.Equal(t, want, got)
	}

	userorgs = c.GetOrganizationsByUserID("u1")
	assert.Empty(t, userorgs)

	wantProjects := map[string]*P{
		"p0": {
			ID:                  "p0",
			OrganizationID:      "o0",
			KubernetesNamespace: "ns0",
		},
		"p1": {
			ID:                  "p1",
			OrganizationID:      "o1",
			KubernetesNamespace: "ns1",
		},
	}
	for id, want := range wantProjects {
		got, ok := c.GetProjectByID(id)
		assert.True(t, ok)
		assert.Equal(t, want, got)

		gots := c.GetProjectsByOrganizationID(want.OrganizationID)
		assert.Len(t, gots, 1)
		assert.Equal(t, *want, gots[0])
	}

	userprojects := c.GetProjectsByUserID("u0")
	assert.Len(t, userprojects, 2)
	userprojectsByProject := map[string]*PU{}
	for _, up := range userprojects {
		userprojectsByProject[up.ProjectID] = &up
	}
	wantPUs := map[string]*PU{
		"p0": {
			Role:      uv1.ProjectRole_PROJECT_ROLE_OWNER,
			ProjectID: "p0",
		},
		"p1": {
			Role:      uv1.ProjectRole_PROJECT_ROLE_MEMBER,
			ProjectID: "p1",
		},
	}
	for projectID, want := range wantPUs {
		got, ok := userprojectsByProject[projectID]
		assert.True(t, ok)
		assert.Equal(t, want, got)
	}

	wantUsers := map[string]*U{
		"u0": {
			ID:         "u0",
			InternalID: "iu0",
			TenantID:   "tid0",
		},
	}
	for id, want := range wantUsers {
		got, ok := c.GetUserByID(id)
		assert.True(t, ok)
		assert.Equal(t, *want, *got)
	}
}

type fakeUserInfoLister struct {
	apikeys      *uv1.ListInternalAPIKeysResponse
	orgs         *uv1.ListInternalOrganizationsResponse
	orgusers     *uv1.ListOrganizationUsersResponse
	projects     *uv1.ListProjectsResponse
	projectusers *uv1.ListProjectUsersResponse
}

func (l *fakeUserInfoLister) ListInternalAPIKeys(ctx context.Context, in *uv1.ListInternalAPIKeysRequest, opts ...grpc.CallOption) (*uv1.ListInternalAPIKeysResponse, error) {
	return l.apikeys, nil
}

func (l *fakeUserInfoLister) ListInternalOrganizations(ctx context.Context, in *uv1.ListInternalOrganizationsRequest, opts ...grpc.CallOption) (*uv1.ListInternalOrganizationsResponse, error) {
	return l.orgs, nil
}

func (l *fakeUserInfoLister) ListOrganizationUsers(ctx context.Context, in *uv1.ListOrganizationUsersRequest, opts ...grpc.CallOption) (*uv1.ListOrganizationUsersResponse, error) {
	return l.orgusers, nil
}

func (l *fakeUserInfoLister) ListProjects(ctx context.Context, in *uv1.ListProjectsRequest, opts ...grpc.CallOption) (*uv1.ListProjectsResponse, error) {
	return l.projects, nil
}

func (l *fakeUserInfoLister) ListProjectUsers(ctx context.Context, in *uv1.ListProjectUsersRequest, opts ...grpc.CallOption) (*uv1.ListProjectUsersResponse, error) {
	return l.projectusers, nil
}

type fakeClusterInfoLister struct {
	clusters *cv1.ListInternalClustersResponse
}

func (l *fakeClusterInfoLister) ListInternalClusters(ctx context.Context, in *cv1.ListInternalClustersRequest, opts ...grpc.CallOption) (*cv1.ListInternalClustersResponse, error) {
	return l.clusters, nil
}
