package main

import (
	"testing"

	"github.com/aveiga/cloud-api-gateway/internal/config"
)

func boolPtr(v bool) *bool {
	return &v
}

func TestSplitRulesByAuth(t *testing.T) {
	rules := []config.RouteRule{
		{Methods: []string{"GET"}, RequireAuth: boolPtr(false)},
		{Methods: []string{"POST"}},
		{Methods: []string{"DELETE"}, RequireAuth: boolPtr(true)},
	}

	publicRules, protectedRules := splitRulesByAuth(rules)
	if len(publicRules) != 1 {
		t.Fatalf("expected 1 public rule, got %d", len(publicRules))
	}
	if len(protectedRules) != 2 {
		t.Fatalf("expected 2 protected rules, got %d", len(protectedRules))
	}
}
