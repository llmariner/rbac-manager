package auth

import (
	"context"
	"errors"

	v1 "github.com/llm-operator/rbac-manager/api/v1"
)

type userInfoKey struct{}

// UserInfo manages the user info.
type UserInfo struct {
	UserID         string
	OrganizationID string
}

func appendUserInfoToContext(ctx context.Context, auth *v1.AuthorizeResponse) context.Context {
	return context.WithValue(ctx, userInfoKey{}, UserInfo{
		UserID:         auth.User.Id,
		OrganizationID: auth.Organization.Id,
	})
}

// ExtractUserInfoFromContext extracts the user info from the context.
func ExtractUserInfoFromContext(ctx context.Context) (*UserInfo, error) {
	info, ok := ctx.Value(userInfoKey{}).(*UserInfo)
	if !ok {
		return nil, errors.New("user info not found")
	}
	return info, nil
}
