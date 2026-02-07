package middleware

import (
	"log"
	"net/http"

	"github.com/aveiga/cloud-api-gateway/internal/config"
)

// RBACMiddleware checks if the authenticated user has the required roles
type RBACMiddleware struct {
	route *config.RouteConfig
}

// NewRBACMiddleware creates a new RBAC middleware for a specific route
func NewRBACMiddleware(route *config.RouteConfig) *RBACMiddleware {
	return &RBACMiddleware{
		route: route,
	}
}

// Handler returns an HTTP handler that checks role permissions
func (m *RBACMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If no roles required, allow access
		if len(m.route.RequiredRoles) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// Get token claims from context
		claims := GetTokenClaims(r)
		if claims == nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Get all roles from token
		userRoles := claims.GetAllRoles()

		// Check if user has required roles
		hasPermission := m.checkRoles(userRoles, m.route.RequiredRoles, m.route.RequireAllRoles)
		if !hasPermission {
			log.Printf("Insufficient permissions for route %s", m.route.Name)
			http.Error(w, "Insufficient permissions", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// checkRoles verifies if user roles satisfy the required roles
// If requireAll is true, user must have ALL required roles (AND logic)
// If requireAll is false, user must have ANY required role (OR logic)
func (m *RBACMiddleware) checkRoles(userRoles []string, requiredRoles []string, requireAll bool) bool {
	if len(requiredRoles) == 0 {
		return true
	}

	// Create a map for O(1) lookup
	roleMap := make(map[string]bool)
	for _, role := range userRoles {
		roleMap[role] = true
	}

	if requireAll {
		// AND logic: user must have all required roles
		for _, required := range requiredRoles {
			if !roleMap[required] {
				return false
			}
		}
		return true
	}

	// OR logic: user must have at least one required role
	for _, required := range requiredRoles {
		if roleMap[required] {
			return true
		}
	}
	return false
}
