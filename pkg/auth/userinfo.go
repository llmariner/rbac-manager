package auth

import (
	"context"

	v1 "github.com/llmariner/rbac-manager/api/v1"
)

type userInfoKey struct{}

// AssignedKubernetesEnv represents the assigned Kubernetes environment.
type AssignedKubernetesEnv struct {
	ClusterID string
	Namespace string
}

// UserInfo manages the user info.
type UserInfo struct {
	UserID                 string
	OrganizationID         string
	ProjectID              string
	AssignedKubernetesEnvs []AssignedKubernetesEnv
	TenantID               string
}

// AppendUserInfoToContext appends the user info to the context.
func AppendUserInfoToContext(ctx context.Context, info UserInfo) context.Context {
	return context.WithValue(ctx, userInfoKey{}, &info)
}

// ExtractUserInfoFromContext extracts the user info from the context.
func ExtractUserInfoFromContext(ctx context.Context) (*UserInfo, bool) {
	info, ok := ctx.Value(userInfoKey{}).(*UserInfo)
	return info, ok
}

func newUserInfoFromAuthorizeResponse(resp *v1.AuthorizeResponse) UserInfo {
	var envs []AssignedKubernetesEnv
	for _, env := range resp.Project.AssignedKubernetesEnvs {
		envs = append(envs, AssignedKubernetesEnv{
			ClusterID: env.ClusterId,
			Namespace: env.Namespace,
		})
	}
	return UserInfo{
		UserID:                 resp.User.Id,
		OrganizationID:         resp.Organization.Id,
		ProjectID:              resp.Project.Id,
		AssignedKubernetesEnvs: envs,
		TenantID:               resp.TenantId,
	}
}
