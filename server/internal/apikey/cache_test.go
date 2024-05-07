package apikey

import (
	"context"
	"testing"

	"github.com/llm-operator/rbac-manager/server/internal/config"
	uv1 "github.com/llm-operator/user-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestCache(t *testing.T) {
	l := &fakeAPIKeyLister{
		resp: &uv1.ListAPIKeysResponse{
			Data: []*uv1.APIKey{
				{
					Id:     "id0",
					Secret: "s0",
				},
				{
					Id:     "id1",
					Secret: "s1",
				},
			},
		},
	}
	c := NewCache(l, &config.DebugConfig{
		APIKeyRole: "role",
	})

	err := c.updateCache(context.Background())
	assert.NoError(t, err)

	want := map[string]*K{
		"s0": {
			Role: "role",
		},
		"s1": {
			Role: "role",
		},
		"s2": nil,
	}

	for k, v := range want {
		got, ok := c.GetAPIKeyBySecret(k)
		if v == nil {
			assert.False(t, ok)
			continue
		}

		assert.True(t, ok)
		assert.Equal(t, v.Role, got.Role)
	}
}

type fakeAPIKeyLister struct {
	resp *uv1.ListAPIKeysResponse
}

func (l *fakeAPIKeyLister) ListAPIKeys(ctx context.Context, in *uv1.ListAPIKeysRequest, opts ...grpc.CallOption) (*uv1.ListAPIKeysResponse, error) {
	return l.resp, nil
}
