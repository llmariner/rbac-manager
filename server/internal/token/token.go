package token

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
