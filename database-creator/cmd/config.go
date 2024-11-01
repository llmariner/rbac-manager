package main

import (
	"fmt"
	"os"

	"github.com/llmariner/common/pkg/db"
	"gopkg.in/yaml.v3"
)

// StorageConfig is the storage configuration.
type StorageConfig struct {
	Config db.Config `yaml:"config"`
}

// Config is the configuration. Follow the format that Dex has in its config.
type Config struct {
	Storage StorageConfig `yaml:"storage"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	return c.Storage.Config.Validate()
}

// parse parses the configuration file at the given path, returning a new
// Config struct.
func parse(path string) (Config, error) {
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
