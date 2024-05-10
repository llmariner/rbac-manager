package auth

import (
	"context"
)

type userInfoKey struct{}

// UserInfo manages the user info.
type UserInfo struct {
	UserID         string
	OrganizationID string
}

func appendUserInfoToContext(ctx context.Context, info UserInfo) context.Context {
	return context.WithValue(ctx, userInfoKey{}, &info)
}

// ExtractUserInfoFromContext extracts the user info from the context.
func ExtractUserInfoFromContext(ctx context.Context) (*UserInfo, bool) {
	info, ok := ctx.Value(userInfoKey{}).(*UserInfo)
	return info, ok
}
