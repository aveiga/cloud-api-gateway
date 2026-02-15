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
