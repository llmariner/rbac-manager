package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the configuration.
type Config struct {
	InternalGRPCPort int `yaml:"internalGrpcPort"`

	IssuerURL string `yaml:"issuerUrl"`

	APIKeyCacheConfig APIKeyCacheConfig `yaml:"apiKeyCache"`

	Debug DebugConfig `yaml:"debug"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.InternalGRPCPort <= 0 {
		return fmt.Errorf("internalGrpcPort must be greater than 0")
	}
	if c.IssuerURL == "" {
		return fmt.Errorf("issuerUrl must be set")
	}
	if err := c.APIKeyCacheConfig.validate(); err != nil {
		return fmt.Errorf("apiKeyCache: %s", err)
	}
	return nil
}

// DebugConfig specifies the debug configurations.
type DebugConfig struct {
	// UserOrgMap maps a registered user(email) to an organization name.
	UserOrgMap map[string]string `yaml:"userOrgMap"`
	// OrgRoleMap maps an organization name to a role name.
	OrgRoleMap map[string]string `yaml:"orgRoleMap"`
	// RoleScopesMap maps a role name to a list of scopes.
	RoleScopesMap map[string][]string `yaml:"roleScopesMap"`

	// APIKeyRole is the role name associated with the API key.
	APIKeyRole string `yaml:"apiKeyRole"`
}

// APIKeyCacheConfig is the API key cache configuration.
type APIKeyCacheConfig struct {
	SyncInterval                  time.Duration `yaml:"syncInterval"`
	UserManagerServerInternalAddr string        `yaml:"userManagerServerInternalAddr"`
}

func (c *APIKeyCacheConfig) validate() error {
	if c.SyncInterval <= 0 {
		return fmt.Errorf("syncInterval must be greater than 0")
	}
	if c.UserManagerServerInternalAddr == "" {
		return fmt.Errorf("userManagerServerInternalAddr must be set")
	}
	return nil
}

// Parse parses the configuration file at the given path, returning a new
// Config struct.
func Parse(path string) (Config, error) {
	var config Config
	b, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("config: read: %s", err)
	}
	if err = yaml.Unmarshal(b, &config); err != nil {
		return config, fmt.Errorf("config: unmarshal: %s", err)
	}
	return config, nil
}
