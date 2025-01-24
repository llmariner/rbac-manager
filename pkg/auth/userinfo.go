package auth

import (
	"context"

	v1 "github.com/llmariner/rbac-manager/api/v1"
)

type userInfoKey struct{}

// AssignedKubernetesEnv represents the assigned Kubernetes environment.
type AssignedKubernetesEnv struct {
	ClusterID   string
	ClusterName string
	Namespace   string
}

// UserInfo manages the user info.
type UserInfo struct {
	UserID                 string
	InternalUserID         string
	OrganizationID         string
	OrganizationTitle      string
	ProjectID              string
	ProjectTitle           string
	AssignedKubernetesEnvs []AssignedKubernetesEnv
	TenantID               string

	// APIKeyID is the ID of the API key. It is set only when the user is authenticated with an API key.
	APIKeyID string
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
			ClusterID:   env.ClusterId,
			ClusterName: env.ClusterName,
			Namespace:   env.Namespace,
		})
	}
	return UserInfo{
		UserID:                 resp.User.Id,
		InternalUserID:         resp.User.InternalId,
		OrganizationID:         resp.Organization.Id,
		OrganizationTitle:      resp.Organization.Title,
		ProjectID:              resp.Project.Id,
		ProjectTitle:           resp.Project.Title,
		APIKeyID:               resp.ApiKeyId,
		AssignedKubernetesEnvs: envs,
		TenantID:               resp.TenantId,
	}
}
