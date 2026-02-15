---
name: improve-unit-tests
description: Improves unit tests by ensuring >90% code coverage and reviewing test quality. Rejects low-value tests that exist only for coverage. Use when improving tests, adding tests, reviewing coverage, or when the user asks for test quality or coverage improvements.
---

# Improve Unit Tests

**Model preference:** Use GPT-5.3 Codex model when applying this skill.

## Goals

1. Achieve and maintain **>90% code coverage**
2. Ensure tests are **valuable** (catch bugs, document behavior, enable refactoring)
3. Avoid tests that exist only to satisfy coverage metrics

## Workflow

### 1. Measure coverage

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Identify packages below 90%.

### 2. Quality review before adding tests

Before adding any test, ask:

- **Does it test real behavior?** Not implementation details.
- **Would it catch a regression?** If the code breaks, would this test fail?
- **Is it readable?** Clear test name, arrange/act/assert structure.
- **Is it independent?** No hidden side effects or order dependencies.

### 3. Valuable vs low-value tests

| Valuable | Low-value |
|----------|-----------|
| Tests error paths and edge cases | Tests only the happy path |
| Tests business logic and invariants | Tests trivial getters/setters |
| Tests integration points | Tests that mock everything |
| Assertions on meaningful outcomes | Assertions on internal state |

### 4. When to remove or refactor tests

- Remove tests that only assert coverage without adding confidence.
- Refactor tests that are brittle (break on refactors that don't change behavior).
- Merge tests that duplicate the same scenario.

### 5. Go test checklist

- [ ] Coverage ≥90% per package
- [ ] Each test has a descriptive name (e.g. `TestLoadRejectsRouteWithoutRules`)
- [ ] Tests use table-driven style where appropriate
- [ ] Error cases are tested
- [ ] No tests that only assert `!= nil` without meaningful checks

## Example: valuable test

```go
// ✅ GOOD: Tests real validation behavior
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
```

## Example: low-value test

```go
// ❌ BAD: Tests nothing meaningful
func TestConfigHasName(t *testing.T) {
    c := Config{Server: ServerConfig{Port: 8080}}
    if c.Server.Port != 8080 {
        t.Fail()
    }
}
```
