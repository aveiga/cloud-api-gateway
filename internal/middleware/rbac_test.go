package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aveiga/cloud-api-gateway/internal/auth"
	"github.com/aveiga/cloud-api-gateway/internal/config"
)

func requestWithRoles(roles []string) *http.Request {
	req := httptest.NewRequest("GET", "/api/users", nil)
	claims := &auth.IntrospectionResponse{
		Active:      true,
		RealmAccess: auth.RealmAccess{Roles: roles},
	}
	ctx := context.WithValue(req.Context(), TokenClaimsKey, claims)
	return req.WithContext(ctx)
}

func TestRBACAllowsWhenAnyRuleMatches(t *testing.T) {
	mw := NewRBACMiddleware("users", []config.RouteRule{
		{Methods: []string{"GET"}, RequiredRoles: []string{"admin"}, RequireAllRoles: true},
		{Methods: []string{"GET"}, RequiredRoles: []string{"editor"}, RequireAllRoles: true},
	})

	rec := httptest.NewRecorder()
	nextCalled := false
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	handler.ServeHTTP(rec, requestWithRoles([]string{"editor"}))

	if !nextCalled || rec.Code != http.StatusNoContent {
		t.Fatalf("expected request to pass, status=%d nextCalled=%v", rec.Code, nextCalled)
	}
}

func TestRBACDeniesWhenNoRuleMatches(t *testing.T) {
	mw := NewRBACMiddleware("users", []config.RouteRule{
		{Methods: []string{"GET"}, RequiredRoles: []string{"admin"}, RequireAllRoles: true},
		{Methods: []string{"GET"}, RequiredRoles: []string{"editor"}, RequireAllRoles: true},
	})

	rec := httptest.NewRecorder()
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	handler.ServeHTTP(rec, requestWithRoles([]string{"viewer"}))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
}

func TestRBACRequiresAuthenticatedClaims(t *testing.T) {
	mw := NewRBACMiddleware("users", []config.RouteRule{
		{Methods: []string{"GET"}, RequiredRoles: []string{"admin"}, RequireAllRoles: true},
	})

	rec := httptest.NewRecorder()
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest("GET", "/api/users", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}
