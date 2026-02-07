package router

import (
	"net/http"
	"strings"

	"github.com/aveiga/cloud-api-gateway/internal/config"
)

// Router matches incoming requests against configured routes
type Router struct {
	routes []*config.RouteConfig
}

// NewRouter creates a new router with the given routes
func NewRouter(routes []config.RouteConfig) *Router {
	// Convert slice of values to slice of pointers for efficient access
	routePtrs := make([]*config.RouteConfig, len(routes))
	for i := range routes {
		routePtrs[i] = &routes[i]
	}

	return &Router{
		routes: routePtrs,
	}
}

// MatchRoute finds the first route that matches the request path and method
// Returns nil if no route matches
func (r *Router) MatchRoute(req *http.Request) *config.RouteConfig {
	path := req.URL.Path
	method := strings.ToUpper(req.Method)

	for _, route := range r.routes {
		// Check if path matches regex pattern (case-insensitive matching via compiled pattern)
		if !route.CompiledPattern.MatchString(path) {
			continue
		}

		// Check if method matches (empty methods list means all methods allowed)
		if len(route.Methods) > 0 {
			methodMatched := false
			for _, m := range route.Methods {
				if m == method {
					methodMatched = true
					break
				}
			}
			if !methodMatched {
				continue
			}
		}

		// Found a match
		return route
	}

	return nil
}
