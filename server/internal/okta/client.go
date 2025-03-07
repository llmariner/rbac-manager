package okta

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/llmariner/rbac-manager/server/internal/token"
)

var _ token.Client = &DefaultClient{}

// ClientOpts are options for NewDefautlClient
type ClientOpts struct {
	Refresh time.Duration
}

// NewDefaultClient returns a new default client.
func NewDefaultClient(ctx context.Context, url string, opts ClientOpts) (*DefaultClient, error) {
	var refreshOpts []jwk.AutoRefreshOption
	if opts.Refresh > 0 {
		refreshOpts = append(refreshOpts, jwk.WithRefreshInterval(opts.Refresh))
	}
	ar := jwk.NewAutoRefresh(ctx)
	ar.Configure(url, refreshOpts...)

	// Perform an initial token refresh so the keys are cached.
	_, err := ar.Refresh(ctx, url)
	if err != nil {
		return nil, err
	}

	return &DefaultClient{
		ctx: ctx,
		url: url,
		ar:  ar,
	}, nil
}

// DefaultClient is the default Okta client.
type DefaultClient struct {
	ctx context.Context
	url string
	ar  *jwk.AutoRefresh
}

// TokenIntrospect introspects the given token.
func (c *DefaultClient) TokenIntrospect(tokenStr string) (*token.Introspection, error) {
	claims, err := c.getClaimsFromAccessToken(tokenStr)
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
	fmt.Printf("Found email[%s], userID[%s] from claims %+v\n", email, userID, claims)

	return &token.Introspection{
		Active:  true,
		Subject: userID,
		Extra: token.IntrospectionExtra{
			Email: email,
		},
	}, nil
}

// getClaimsFromAccessToken gets the claims from the JWT access token.
func (c *DefaultClient) getClaimsFromAccessToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := c.validate(tokenStr)
	if err != nil {
		return nil, err

	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("unexpected form of claims: %s", err)
	}
	return claims, nil
}

// validate validates the incoming token string against the public key.
func (c *DefaultClient) validate(tokenStr string) (*jwt.Token, error) {
	set, err := c.ar.Fetch(c.ctx, c.url)
	if err != nil {
		return nil, err
	}

	// Validate with all keys until we find a match.
	for i := 0; i < set.Len(); i++ {
		key, ok := set.Get(i)
		if !ok {
			return nil, fmt.Errorf("idx %d out of range (keys = %d)", i, set.Len())
		}

		var rawKey interface{}
		if err = key.Raw(&rawKey); err != nil {
			return nil, fmt.Errorf("raw: %s", err)
		}

		switch k := rawKey.(type) {
		case *rsa.PublicKey:
		default:
			return nil, fmt.Errorf("unknown key type: %T", k)
		}

		t, _ := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return rawKey, nil
		})

		if t != nil && t.Valid {
			return t, nil
		}
	}

	return nil, fmt.Errorf("no key to validate")
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
