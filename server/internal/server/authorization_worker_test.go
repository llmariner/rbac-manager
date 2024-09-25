package server

import (
	"context"
	"testing"

	v1 "github.com/llmariner/rbac-manager/api/v1"
	"github.com/llmariner/rbac-manager/server/internal/cache"
	"github.com/stretchr/testify/assert"
)

func TestAuthorizeWorker(t *testing.T) {
	tcs := []struct {
		name     string
		req      *v1.AuthorizeWorkerRequest
		clusters map[string]*cache.C
		want     bool
	}{
		{
			name: "authorized with cluster registration key",
			req: &v1.AuthorizeWorkerRequest{
				Token: "rkey0",
			},
			clusters: map[string]*cache.C{
				"rkey0": {
					ID: "c0",
				},
			},
			want: true,
		},
		{
			name: "unauthorized with invalid key",
			req: &v1.AuthorizeWorkerRequest{
				Token: "invalid",
			},
			clusters: map[string]*cache.C{
				"rkey0": {
					ID: "c0",
				},
			},
			want: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			srv := &Server{
				cache: &fakeCacheGetter{
					clusters: tc.clusters,
				},
			}
			resp, err := srv.AuthorizeWorker(context.Background(), tc.req)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, resp.Authorized)
		})
	}
}
