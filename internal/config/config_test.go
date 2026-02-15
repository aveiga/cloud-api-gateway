package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func baseConfig(routes string) string {
	return `
server:
  port: 4010
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
authz:
  introspection_url: "http://keycloak/introspect"
  client_id: "gateway"
  client_secret: "secret"
  timeout: 5s
cache:
  enabled: true
  ttl: 60s
routes:
` + routes
}

func TestLoadRejectsRouteWithoutRules(t *testing.T) {
	cfgPath := writeConfig(t, baseConfig(`
  - name: "users"
    path_pattern: "^/api/users(/.*)?$"
    upstream: "http://users:8080"
`))

	_, err := Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "must define at least one rules entry") {
		t.Fatalf("expected rules validation error, got: %v", err)
	}
}

func TestLoadRejectsRouteLevelMethods(t *testing.T) {
	cfgPath := writeConfig(t, baseConfig(`
  - name: "users"
    path_pattern: "^/api/users(/.*)?$"
    methods: ["GET"]
    upstream: "http://users:8080"
    rules:
      - methods: ["GET"]
        required_roles: ["admin"]
        require_all_roles: true
`))

	_, err := Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "route-level methods/required_roles/require_all_roles") {
		t.Fatalf("expected route-level legacy fields validation error, got: %v", err)
	}
}

func TestLoadNormalizesRuleMethods(t *testing.T) {
	cfgPath := writeConfig(t, baseConfig(`
  - name: "users"
    path_pattern: "^/api/users(/.*)?$"
    upstream: "http://users:8080"
    rules:
      - methods: ["get", "post"]
        required_roles: []
        require_all_roles: true
`))

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	got := cfg.Routes[0].Rules[0].Methods
	if len(got) != 2 || got[0] != "GET" || got[1] != "POST" {
		t.Fatalf("unexpected normalized methods: %v", got)
	}
}

func TestLoadRuleRequireAuthDefaultsToTrue(t *testing.T) {
	cfgPath := writeConfig(t, baseConfig(`
  - name: "users"
    path_pattern: "^/api/users(/.*)?$"
    upstream: "http://users:8080"
    rules:
      - methods: ["GET"]
        required_roles: []
        require_all_roles: true
`))

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Routes[0].Rules[0].RequiresAuth() {
		t.Fatal("expected rule require_auth default to true")
	}
}

func TestLoadRejectsRouteLevelRequireAuth(t *testing.T) {
	cfgPath := writeConfig(t, baseConfig(`
  - name: "health"
    path_pattern: "^/health$"
    upstream: "http://health:8080"
    require_auth: false
    rules:
      - methods: ["GET"]
`))

	_, err := Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "route-level require_auth is not supported") {
		t.Fatalf("expected route-level require_auth validation error, got: %v", err)
	}
}

func TestLoadRejectsUnauthenticatedRuleWithRequiredRoles(t *testing.T) {
	cfgPath := writeConfig(t, baseConfig(`
  - name: "health"
    path_pattern: "^/health$"
    upstream: "http://health:8080"
    rules:
      - methods: ["GET"]
        require_auth: false
        required_roles: ["admin"]
        require_all_roles: true
`))

	_, err := Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "rules with require_auth=false cannot define required_roles") {
		t.Fatalf("expected unauthenticated rule required_roles validation error, got: %v", err)
	}
}

func TestLoadFailsWhenFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil || !strings.Contains(err.Error(), "failed to read config file") {
		t.Fatalf("expected file read error, got: %v", err)
	}
}

func TestLoadRejectsMissingUpstream(t *testing.T) {
	cfgPath := writeConfig(t, baseConfig(`
  - name: "users"
    path_pattern: "^/api/users(/.*)?$"
    rules:
      - methods: ["GET"]
        required_roles: []
        require_all_roles: true
`))
	_, err := Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "upstream") {
		t.Fatalf("expected upstream required error, got: %v", err)
	}
}

func TestLoadRejectsRuleWithEmptyMethods(t *testing.T) {
	cfgPath := writeConfig(t, baseConfig(`
  - name: "users"
    path_pattern: "^/api/users(/.*)?$"
    upstream: "http://users:8080"
    rules:
      - methods: []
        required_roles: []
        require_all_roles: true
`))
	_, err := Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "methods is required") {
		t.Fatalf("expected methods required error, got: %v", err)
	}
}

func TestLoadRejectsInvalidServerPort(t *testing.T) {
	cfgPath := writeConfig(t, `
server:
  port: 0
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
authz:
  introspection_url: "http://keycloak/introspect"
  client_id: "gateway"
  client_secret: "secret"
  timeout: 5s
cache:
  enabled: true
  ttl: 60s
routes:
  - name: "users"
    path_pattern: "^/api/users(/.*)?$"
    upstream: "http://users:8080"
    rules:
      - methods: ["GET"]
        required_roles: []
        require_all_roles: true
`)
	_, err := Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "invalid server port") {
		t.Fatalf("expected invalid port error, got: %v", err)
	}
}

func TestLoadRejectsMissingPathPattern(t *testing.T) {
	cfgPath := writeConfig(t, baseConfig(`
  - name: "users"
    upstream: "http://users:8080"
    rules:
      - methods: ["GET"]
        required_roles: []
        require_all_roles: true
`))
	_, err := Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "path_pattern is required") {
		t.Fatalf("expected path_pattern error, got: %v", err)
	}
}

func TestLoadRejectsInvalidRegex(t *testing.T) {
	cfgPath := writeConfig(t, baseConfig(`
  - name: "users"
    path_pattern: "[invalid(regex"
    upstream: "http://users:8080"
    rules:
      - methods: ["GET"]
        required_roles: []
        require_all_roles: true
`))
	_, err := Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "path_pattern invalid regex") {
		t.Fatalf("expected invalid regex error, got: %v", err)
	}
}

func TestLoadSubstitutesEnvVars(t *testing.T) {
	os.Setenv("TEST_REALM", "myrealm")
	defer os.Unsetenv("TEST_REALM")

	cfgPath := writeConfig(t, `
server:
  port: 4010
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
authz:
  introspection_url: "http://keycloak/realms/${TEST_REALM}/introspect"
  client_id: "gateway"
  client_secret: "secret"
  timeout: 5s
cache:
  enabled: true
  ttl: 60s
routes:
  - name: "users"
    path_pattern: "^/api/users(/.*)?$"
    upstream: "http://users:8080"
    rules:
      - methods: ["GET"]
        required_roles: []
        require_all_roles: true
`)
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Authz.IntrospectionURL != "http://keycloak/realms/myrealm/introspect" {
		t.Fatalf("expected env var substitution, got: %s", cfg.Authz.IntrospectionURL)
	}
}

func TestRouteRuleRequiresAuthWhenExplicitTrue(t *testing.T) {
	trueVal := true
	r := RouteRule{RequireAuth: &trueVal}
	if !r.RequiresAuth() {
		t.Fatal("expected RequiresAuth true when RequireAuth is true")
	}
}

func TestLoadSubstitutesEnvVarWithDefault(t *testing.T) {
	os.Unsetenv("MISSING_VAR")
	defer os.Unsetenv("MISSING_VAR")

	cfgPath := writeConfig(t, `
server:
  port: 4010
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
authz:
  introspection_url: "http://keycloak/realms/${MISSING_VAR:-defaultrealm}/introspect"
  client_id: "gateway"
  client_secret: "secret"
  timeout: 5s
cache:
  enabled: true
  ttl: 60s
routes:
  - name: "users"
    path_pattern: "^/api/users(/.*)?$"
    upstream: "http://users:8080"
    rules:
      - methods: ["GET"]
        required_roles: []
        require_all_roles: true
`)
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Authz.IntrospectionURL != "http://keycloak/realms/defaultrealm/introspect" {
		t.Fatalf("expected default value, got: %s", cfg.Authz.IntrospectionURL)
	}
}

func TestLoadRejectsMissingAuthzIntrospectionURL(t *testing.T) {
	cfgPath := writeConfig(t, `
server:
  port: 4010
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
authz:
  introspection_url: ""
  client_id: "gateway"
  client_secret: "secret"
  timeout: 5s
cache:
  enabled: true
  ttl: 60s
routes:
  - name: "users"
    path_pattern: "^/api/users(/.*)?$"
    upstream: "http://users:8080"
    rules:
      - methods: ["GET"]
        required_roles: []
        require_all_roles: true
`)
	_, err := Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "introspection_url") {
		t.Fatalf("expected introspection_url required error, got: %v", err)
	}
}

func TestLoadAcceptsPatternWithExistingCaseInsensitiveFlag(t *testing.T) {
	cfgPath := writeConfig(t, baseConfig(`
  - name: "users"
    path_pattern: "(?i)^/api/users(/.*)?$"
    upstream: "http://users:8080"
    rules:
      - methods: ["GET"]
        required_roles: []
        require_all_roles: true
`))
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Routes[0].CompiledPattern == nil {
		t.Fatal("expected compiled pattern")
	}
}
