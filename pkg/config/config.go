package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the root configuration
type Config struct {
	Version    string              `yaml:"version"`
	Global     GlobalConfig        `yaml:"global"`
	Registries map[string]Registry `yaml:"registries"`
	SyncRules  []SyncRule          `yaml:"sync_rules"`
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	Concurrency int           `yaml:"concurrency"`
	Retry       RetryConfig   `yaml:"retry"`
	Timeout     time.Duration `yaml:"timeout"`
}

// RetryConfig contains retry settings
type RetryConfig struct {
	MaxAttempts     int           `yaml:"max_attempts"`
	InitialInterval time.Duration `yaml:"initial_interval"`
	MaxInterval     time.Duration `yaml:"max_interval"`
}

// Registry represents a container registry
type Registry struct {
	URL       string        `yaml:"url"`
	Username  string        `yaml:"username"`
	Password  string        `yaml:"password"`
	Insecure  bool          `yaml:"insecure"`
	RateLimit RateLimitInfo `yaml:"ratelimit,omitempty"`
}

// RateLimitInfo contains rate limiting settings
type RateLimitInfo struct {
	QPS int `yaml:"qps"`
}

// SyncRule represents a single sync task
type SyncRule struct {
	Name          string         `yaml:"name"`
	Source        SourceConfig   `yaml:"source"`
	Target        TargetConfig   `yaml:"target"`
	Tags          TagFilter      `yaml:"tags"`
	Architectures []string       `yaml:"architectures"`
	Enabled       bool           `yaml:"enabled"`
}

// SourceConfig represents source registry configuration
type SourceConfig struct {
	Registry   string `yaml:"registry"`
	Repository string `yaml:"repository"`
}

// TargetConfig represents target registry configuration
type TargetConfig struct {
	Registry   string `yaml:"registry"`
	Repository string `yaml:"repository"`
}

// TagFilter contains tag filtering rules
type TagFilter struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
	Latest  int      `yaml:"latest"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	var config Config
	if err := yaml.Unmarshal([]byte(expanded), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	if config.Global.Concurrency == 0 {
		config.Global.Concurrency = 3
	}
	if config.Global.Retry.MaxAttempts == 0 {
		config.Global.Retry.MaxAttempts = 3
	}
	if config.Global.Retry.InitialInterval == 0 {
		config.Global.Retry.InitialInterval = 1 * time.Second
	}
	if config.Global.Retry.MaxInterval == 0 {
		config.Global.Retry.MaxInterval = 30 * time.Second
	}
	if config.Global.Timeout == 0 {
		config.Global.Timeout = 10 * time.Minute
	}

	// Validate
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Registries) == 0 {
		return fmt.Errorf("no registries defined")
	}

	for name, reg := range c.Registries {
		if reg.URL == "" {
			return fmt.Errorf("registry %s: URL is required", name)
		}
	}

	for i, rule := range c.SyncRules {
		if rule.Name == "" {
			return fmt.Errorf("sync rule %d: name is required", i)
		}
		if _, ok := c.Registries[rule.Source.Registry]; !ok {
			return fmt.Errorf("sync rule %s: source registry %s not found", rule.Name, rule.Source.Registry)
		}
		if _, ok := c.Registries[rule.Target.Registry]; !ok {
			return fmt.Errorf("sync rule %s: target registry %s not found", rule.Name, rule.Target.Registry)
		}
		if rule.Source.Repository == "" {
			return fmt.Errorf("sync rule %s: source repository is required", rule.Name)
		}
		if rule.Target.Repository == "" {
			return fmt.Errorf("sync rule %s: target repository is required", rule.Name)
		}

		// Validate regex patterns
		for _, pattern := range rule.Tags.Include {
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("sync rule %s: invalid include pattern %s: %w", rule.Name, pattern, err)
			}
		}
		for _, pattern := range rule.Tags.Exclude {
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("sync rule %s: invalid exclude pattern %s: %w", rule.Name, pattern, err)
			}
		}
	}

	return nil
}

// GetRegistry returns a registry configuration by name
func (c *Config) GetRegistry(name string) (Registry, error) {
	reg, ok := c.Registries[name]
	if !ok {
		return Registry{}, fmt.Errorf("registry %s not found", name)
	}
	return reg, nil
}

// GetEnabledRules returns all enabled sync rules
func (c *Config) GetEnabledRules() []SyncRule {
	var rules []SyncRule
	for _, rule := range c.SyncRules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	return rules
}

// NormalizeRegistryURL normalizes a registry URL
func NormalizeRegistryURL(url string) string {
	// Remove trailing slash
	url = strings.TrimRight(url, "/")

	// Add https:// if no scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	return url
}
