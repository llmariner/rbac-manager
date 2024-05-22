package auth

import (
	"context"

	v1 "github.com/llm-operator/rbac-manager/api/v1"
)

type userInfoKey struct{}

// UserInfo manages the user info.
type UserInfo struct {
	UserID               string
	OrganizationID       string
	ProjectID            string
	KuberenetesNamespace string
}

func appendUserInfoToContext(ctx context.Context, info UserInfo) context.Context {
	return context.WithValue(ctx, userInfoKey{}, &info)
}

// ExtractUserInfoFromContext extracts the user info from the context.
func ExtractUserInfoFromContexta(ctx context.Context) (*UserInfo, bool) {
	info, ok := ctx.Value(userInfoKey{}).(*UserInfo)
	return info, ok
}

func newUserInfoFromAuthorizeResponse(resp *v1.AuthorizeResponse) UserInfo {
	return UserInfo{
		UserID:               resp.User.Id,
		OrganizationID:       resp.Organization.Id,
		ProjectID:            resp.Project.Id,
		KuberenetesNamespace: resp.Project.KubernetesNamespace,
	}
}
