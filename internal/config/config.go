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
	Authz AuthzConfig `yaml:"authz"`
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

// AuthzConfig holds authz (token introspection) connection settings
type AuthzConfig struct {
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

// RouteRule defines method, authentication, and role requirements.
type RouteRule struct {
	Methods         []string `yaml:"methods"`
	RequireAuth     *bool    `yaml:"require_auth"` // nil defaults to true
	RequiredRoles   []string `yaml:"required_roles"`
	RequireAllRoles bool     `yaml:"require_all_roles"`
}

// RouteConfig represents a single route configuration
type RouteConfig struct {
	Name              string `yaml:"name"`
	PathPattern       string `yaml:"path_pattern"`
	CompiledPattern   *regexp.Regexp
	Methods           []string    `yaml:"methods"`
	Upstream          string      `yaml:"upstream"`
	StripPrefix       string      `yaml:"strip_prefix"`
	RequiredRoles     []string    `yaml:"required_roles"`
	RequireAllRoles   bool        `yaml:"require_all_roles"`
	LegacyRequireAuth *bool       `yaml:"require_auth"` // disallowed at route level; use rules[].require_auth
	Rules             []RouteRule `yaml:"rules"`
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

	// Validate authz config
	if c.Authz.IntrospectionURL == "" {
		return fmt.Errorf("authz.introspection_url is required")
	}
	if c.Authz.ClientID == "" {
		return fmt.Errorf("authz.client_id is required")
	}
	if c.Authz.ClientSecret == "" {
		return fmt.Errorf("authz.client_secret is required")
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

		// Compile regex pattern with case-insensitive matching
		// Add (?i) flag at the beginning if not already present
		pattern := route.PathPattern
		if !strings.HasPrefix(pattern, "(?i)") && !strings.HasPrefix(pattern, "(?i:") {
			pattern = "(?i)" + pattern
		}
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("route[%d].path_pattern invalid regex: %w", i, err)
		}
		route.CompiledPattern = compiled

		if len(route.Methods) > 0 || len(route.RequiredRoles) > 0 || route.RequireAllRoles {
			return fmt.Errorf("route[%d]: route-level methods/required_roles/require_all_roles are not supported; use rules[]", i)
		}
		if route.LegacyRequireAuth != nil {
			return fmt.Errorf("route[%d]: route-level require_auth is not supported; use rules[].require_auth", i)
		}
		if len(route.Rules) == 0 {
			return fmt.Errorf("route[%d]: routes must define at least one rules entry", i)
		}
		for j := range route.Rules {
			rule := &route.Rules[j]
			if len(rule.Methods) == 0 {
				return fmt.Errorf("route[%d].rules[%d]: methods is required", i, j)
			}
			for k := range rule.Methods {
				rule.Methods[k] = strings.ToUpper(rule.Methods[k])
			}
			if !rule.RequiresAuth() && len(rule.RequiredRoles) > 0 {
				return fmt.Errorf("route[%d].rules[%d]: rules with require_auth=false cannot define required_roles", i, j)
			}
		}
	}

	return nil
}

// RequiresAuth returns true if authentication is required for this rule.
// Defaults to true if require_auth is not specified.
func (r *RouteRule) RequiresAuth() bool {
	if r.RequireAuth == nil {
		return true // Default to requiring auth if not specified
	}
	return *r.RequireAuth
}
