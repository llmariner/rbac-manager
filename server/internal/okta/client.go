package okta

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/llmariner/rbac-manager/server/internal/token"
)

var _ token.Client = &defaultClient{}

// NewDefaultClient returns a new default client.
func NewDefaultClient() token.Client {
	return &defaultClient{}
}

type defaultClient struct{}

// TokenIntrospect introspects the given token.
func (c *defaultClient) TokenIntrospect(tokenStr string) (*token.Introspection, error) {
	if !strings.HasPrefix(tokenStr, "Bearer ") {
		return nil, fmt.Errorf("unexpected form of auth header")
	}
	accessToken := tokenStr[7:]
	claims, err := getClaimsFromAccessToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("unexpected form of claims: %s", err)
	}

	email, err := getEmail(claims)
	if err != nil {
		return nil, fmt.Errorf("could not get email claim: %s", err)
	}

	userID, err := getUserID(claims)
	if err != nil {
		return nil, fmt.Errorf("could not get user ID: %s", err)
	}

	return &token.Introspection{
		Active:  true,
		Subject: userID,
		Extra: token.IntrospectionExtra{
			Email: email,
		},
	}, nil
}

// getClaimsFromAccessToken gets the claims from the JWT access token.
func getClaimsFromAccessToken(accessToken string) (jwt.MapClaims, error) {
	// Decode the JWT token. Pass nil to keyFunc to skip
	// validation. The access token should have already been
	// validated by KONG gateway.
	token, err := jwt.Parse(accessToken, nil)
	if err != nil {
		// Return the error if the error is not an expected validation error.
		ve, ok := err.(*jwt.ValidationError)
		if !ok {
			return nil, fmt.Errorf("failed to parse: %s", err)
		}
		if ve.Errors != jwt.ValidationErrorUnverifiable {
			return nil, fmt.Errorf("unexpected validation error: %s", err)
		}
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("unexpected form of claims: %s", err)
	}
	return claims, nil
}

// getUserID gets the userID from the JWT claims.
// We get userID only when the claims contain "uid" in access token, or "sub" in ID token.
// Claims contain "uid" only when requests are made by end users (not by Cluster Controller).
func getUserID(claims jwt.MapClaims) (string, error) {
	userID, ok := claims["uid"]
	if !ok {
		userID, ok = claims["sub"]
		if !ok {
			return "", fmt.Errorf("no \"uid\" or \"sub\" claim found in the token")
		}
	}
	v, ok := userID.(string)
	if !ok {
		return "", fmt.Errorf("unexpected type %T for the \"user_id\" claim %v", userID, userID)
	}
	return v, nil
}

// getEmail gets the email from the JWT claims.
// We get email only when the claims contain "sub" in ID token.
// Claims contain no "sub" only when requests are made by Cluster Controller).
func getEmail(claims jwt.MapClaims) (string, error) {
	email, ok := claims["sub"]
	if !ok {
		return "", nil
	}

	v, ok := email.(string)
	if !ok {
		return "", fmt.Errorf("unexpected type %T for the \"sub\" claim %v", email, email)
	}
	return v, nil
}
