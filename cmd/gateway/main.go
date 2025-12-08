package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aveiga/cloud-api-gateway/internal/auth"
	"github.com/aveiga/cloud-api-gateway/internal/config"
	"github.com/aveiga/cloud-api-gateway/internal/middleware"
	"github.com/aveiga/cloud-api-gateway/internal/proxy"
	"github.com/aveiga/cloud-api-gateway/internal/router"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file (or set CONFIG_PATH env var)")
	flag.Parse()

	// Use environment variable if flag not provided
	if *configPath == "" {
		*configPath = os.Getenv("CONFIG_PATH")
	}

	if *configPath == "" {
		log.Fatal("Configuration file path required (use -config flag or CONFIG_PATH env var)")
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize components
	keycloakClient := auth.NewClient(&cfg.Keycloak, cfg.Cache.Enabled, cfg.Cache.TTL)
	routeRouter := router.NewRouter(cfg.Routes)
	authMW := middleware.NewAuthMiddleware(keycloakClient)

	// Create HTTP handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Match route
		matchedRoute := routeRouter.MatchRoute(r)
		if matchedRoute == nil {
			http.Error(w, "Route not found", http.StatusNotFound)
			return
		}

		// Create proxy for this route
		routeProxy, err := proxy.NewProxy(matchedRoute)
		if err != nil {
			log.Printf("Failed to create proxy for route %s: %v", matchedRoute.Name, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Compose middleware chain conditionally
		// If route has required roles, apply Auth -> RBAC -> Proxy
		// Otherwise, just Proxy (public route)
		var chain http.Handler = routeProxy
		
		if len(matchedRoute.RequiredRoles) > 0 {
			// Route requires authentication and authorization
			rbacMW := middleware.NewRBACMiddleware(matchedRoute)
			chain = authMW.Handler(rbacMW.Handler(routeProxy))
		}
		
		chain.ServeHTTP(w, r)
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting API Gateway on port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

