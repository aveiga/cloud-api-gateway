package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the root configuration structure
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Keycloak KeycloakConfig `yaml:"keycloak"`
	Cache    CacheConfig    `yaml:"cache"`
	Routes   []RouteConfig  `yaml:"routes"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// KeycloakConfig holds Keycloak connection settings
type KeycloakConfig struct {
	IntrospectionURL string        `yaml:"introspection_url"`
	ClientID         string        `yaml:"client_id"`
	ClientSecret     string        `yaml:"client_secret"`
	Timeout          time.Duration `yaml:"timeout"`
}

// CacheConfig holds token cache settings
type CacheConfig struct {
	Enabled bool          `yaml:"enabled"`
	TTL     time.Duration `yaml:"ttl"`
}

// RouteConfig represents a single route configuration
type RouteConfig struct {
	Name            string   `yaml:"name"`
	PathPattern     string   `yaml:"path_pattern"`
	CompiledPattern *regexp.Regexp
	Methods         []string `yaml:"methods"`
	Upstream        string   `yaml:"upstream"`
	StripPrefix     string   `yaml:"strip_prefix"`
	RequiredRoles   []string `yaml:"required_roles"`
	RequireAllRoles bool     `yaml:"require_all_roles"`
}

// Load reads and parses the YAML configuration file
func Load(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Substitute environment variables
	content := substituteEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate and pre-compile regex patterns
	if err := cfg.validateAndCompile(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &cfg, nil
}

// substituteEnvVars replaces ${VAR} or ${VAR:-default} patterns with environment variable values
func substituteEnvVars(content string) string {
	// Pattern: ${VAR} or ${VAR:-default}
	re := regexp.MustCompile(`\$\{([^}:]+)(?::-([^}]*))?\}`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		varName := parts[1]
		defaultValue := ""
		if len(parts) > 2 {
			defaultValue = parts[2]
		}

		value := os.Getenv(varName)
		if value == "" {
			return defaultValue
		}
		return value
	})
}

// validateAndCompile validates configuration and pre-compiles regex patterns
func (c *Config) validateAndCompile() error {
	// Validate server config
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate Keycloak config
	if c.Keycloak.IntrospectionURL == "" {
		return fmt.Errorf("keycloak.introspection_url is required")
	}
	if c.Keycloak.ClientID == "" {
		return fmt.Errorf("keycloak.client_id is required")
	}
	if c.Keycloak.ClientSecret == "" {
		return fmt.Errorf("keycloak.client_secret is required")
	}

	// Validate and compile route patterns
	for i := range c.Routes {
		route := &c.Routes[i]
		if route.PathPattern == "" {
			return fmt.Errorf("route[%d].path_pattern is required", i)
		}
		if route.Upstream == "" {
			return fmt.Errorf("route[%d].upstream is required", i)
		}

		// Compile regex pattern
		compiled, err := regexp.Compile(route.PathPattern)
		if err != nil {
			return fmt.Errorf("route[%d].path_pattern invalid regex: %w", i, err)
		}
		route.CompiledPattern = compiled

		// Normalize methods to uppercase
		for j := range route.Methods {
			route.Methods[j] = strings.ToUpper(route.Methods[j])
		}
	}

	return nil
}

