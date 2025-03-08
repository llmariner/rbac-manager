package dex

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/llmariner/rbac-manager/server/internal/token"
)

// NewDefaultClient returns a new default client.
func NewDefaultClient(dexServerAddr string) token.Client {
	return &defaultClient{dexServerAddr: dexServerAddr}
}

type defaultClient struct {
	dexServerAddr string
}

// TokenIntrospect introspects the given token.
func (c *defaultClient) TokenIntrospect(tokenStr string) (*token.Introspection, error) {
	resp, err := http.PostForm(
		fmt.Sprintf("http://%s/v1/dex/token/introspect", c.dexServerAddr),
		url.Values{"token": {tokenStr}})
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
	var is token.Introspection
	if err := json.NewDecoder(resp.Body).Decode(&is); err != nil {
		return nil, err
	}
	return &is, nil
}
