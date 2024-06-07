package auth

import (
	"context"

	v1 "github.com/llm-operator/rbac-manager/api/v1"
)

type clusterInfoKey struct{}

// ClusterInfo manages the cluster info.
type ClusterInfo struct {
	ClusterID string
	TenantID  string
}

// AppendClusterInfoToContext appends the cluster info to the context.
func AppendClusterInfoToContext(ctx context.Context, info ClusterInfo) context.Context {
	return context.WithValue(ctx, clusterInfoKey{}, &info)
}

// ExtractClusterInfoFromContext extracts the cluster info from the context.
func ExtractClusterInfoFromContext(ctx context.Context) (*ClusterInfo, bool) {
	info, ok := ctx.Value(clusterInfoKey{}).(*ClusterInfo)
	return info, ok
}

func newClusterInfoFromAuthorizeResponse(resp *v1.AuthorizeWorkerResponse) ClusterInfo {
	return ClusterInfo{
		ClusterID: resp.Cluster.Id,
		TenantID:  resp.TenantId,
	}
}
