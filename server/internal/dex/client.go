package dex

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

// Client is the interface for the token introspection client.
type Client interface {
	TokenIntrospect(token string) (*Introspection, error)
}

// Introspection is the response from the token introspection endpoint.
type Introspection struct {
	Active  bool               `json:"active"`
	Subject string             `json:"sub"`
	Extra   IntrospectionExtra `json:"ext,omitempty"`
}

// IntrospectionExtra is the extra fields that can be returned in the token introspection response.
type IntrospectionExtra struct {
	Email         string `json:"email,omitempty"`
	EmailVerified *bool  `json:"email_verified,omitempty"`
}

// NewDefaultClient returns a new default client.
func NewDefaultClient(dexServerAddr string) Client {
	return &defaultClient{dexServerAddr: dexServerAddr}
}

type defaultClient struct {
	dexServerAddr string
}

// TokenIntrospect introspects the given token.
func (c *defaultClient) TokenIntrospect(token string) (*Introspection, error) {
	resp, err := http.PostForm(
		fmt.Sprintf("http://%s/v1/dex/token/introspect", c.dexServerAddr),
		url.Values{"token": {token}})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to introspect token: %s", resp.Status)
	}

	var is Introspection
	if err := json.NewDecoder(resp.Body).Decode(&is); err != nil {
		return nil, err
	}
	return &is, nil
}
