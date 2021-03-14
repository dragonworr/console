package git

import (
	"flag"
	"fmt"
	"time"
)

// Config for Git Service
type Config struct {
	Enabled bool `koanf:"enabled"`

	// AllowedFileExtensions specifies file extensions that shall be picked up. If at least one is specified all other
	// file extensions will be ignored.
	AllowedFileExtensions []string `koanf:"-"`

	// Max file size which will be considered. Files exceeding this size will be ignored and logged.
	MaxFileSize int64 `koanf:"-"`

	// Whether or not to use the filename or the full filepath as key in the map
	IndexByFullFilepath bool `koanf:"-"`

	// RefreshInterval specifies how often the repository shall be pulled to check for new changes.
	RefreshInterval time.Duration `koanf:"refreshInterval"`

	// Repository that contains markdown files that document a Kafka topic.
	Repository RepositoryConfig `koanf:"repository"`

	// Authentication Configs
	BasicAuth BasicAuthConfig `koanf:"basicAuth"`
	SSH       SSHConfig       `koanf:"ssh"`
}

// RegisterFlagsWithPrefix for all (sub)configs
func (c *Config) RegisterFlagsWithPrefix(f *flag.FlagSet, prefix string) {
	c.BasicAuth.RegisterFlagsWithPrefix(f, prefix)
	c.SSH.RegisterFlagsWithPrefix(f, prefix)
}

// Validate all root and child config structs
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.RefreshInterval == 0 {
		return fmt.Errorf("git config is enabled but refresh interval is set to 0 (disabled)")
	}

	return c.Repository.Validate()
}

// SetDefaults for all root and child config structs
func (c *Config) SetDefaults() {
	c.RefreshInterval = time.Minute
	c.MaxFileSize = 500 * 1000 // 500KB
	c.IndexByFullFilepath = false
}
