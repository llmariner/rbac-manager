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
	MonitoringPort   int `yaml:"monitoringPort"`

	DexServerAddr string `yaml:"dexServerAddr"`

	CacheConfig CacheConfig `yaml:"cache"`

	// RoleScopesMap maps a role name to a list of scopes.
	RoleScopesMap map[string][]string `yaml:"roleScopesMap"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.InternalGRPCPort <= 0 {
		return fmt.Errorf("internalGrpcPort must be greater than 0")
	}
	if c.MonitoringPort <= 0 {
		return fmt.Errorf("monitoringPort must be greater than 0")
	}
	if c.DexServerAddr == "" {
		return fmt.Errorf("dexServerAddr must be set")
	}
	if err := c.CacheConfig.validate(); err != nil {
		return fmt.Errorf("cache: %s", err)
	}
	return nil
}

// CacheConfig is the API key cache configuration.
type CacheConfig struct {
	SyncInterval                     time.Duration `yaml:"syncInterval"`
	UserManagerServerInternalAddr    string        `yaml:"userManagerServerInternalAddr"`
	ClusterManagerServerInternalAddr string        `yaml:"clusterManagerServerInternalAddr"`
}

func (c *CacheConfig) validate() error {
	if c.SyncInterval <= 0 {
		return fmt.Errorf("syncInterval must be greater than 0")
	}
	if c.UserManagerServerInternalAddr == "" {
		return fmt.Errorf("userManagerServerInternalAddr must be set")
	}
	if c.ClusterManagerServerInternalAddr == "" {
		return fmt.Errorf("clusterManagerServerInternalAddr must be set")
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
