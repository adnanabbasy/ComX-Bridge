package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/commatea/ComX-Bridge/pkg/api/middleware"
	"github.com/commatea/ComX-Bridge/pkg/core"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents the REST API server.
type Server struct {
	engine *core.Engine
	srv    *http.Server
	config ServerConfig
}

// ServerConfig holds API server configuration.
type ServerConfig struct {
	Port int
}

// NewServer creates a new REST API server.
func NewServer(engine *core.Engine, config ServerConfig) *Server {
	return &Server{
		engine: engine,
		config: config,
	}
}

// Start starts the API server.
func (s *Server) Start() error {
	r := mux.NewRouter()

	// Register routes
	s.registerRoutes(r)

	// Apply Middleware
	if s.engine.Config().API.Auth.Enabled {
		config := s.engine.Config().API.Auth
		var keys []string
		// Legacy support check if needed, but we replaced struct. Assuming config load handles it or clean slate.
		for _, u := range config.Users {
			keys = append(keys, u.Key)
		}

		auth := middleware.NewAPIKeyAuth(keys, config.JWTSecret)
		r.Use(auth.Handler)
		fmt.Println("API Authentication enabled (JWT + API Key)")
	}

	// Create address
	addr := fmt.Sprintf(":%d", s.config.Port)
	if s.config.Port == 0 {
		addr = ":8080"
	}

	s.srv = &http.Server{
		Addr:    addr,
		Handler: r,
	}

	fmt.Printf("API Server listening on %s\n", addr)

	// Run server in goroutine
	go func() {
		apiConfig := s.engine.Config().API
		if apiConfig.TLS.Enabled {
			fmt.Println("API Server starting in HTTPS mode")
			if err := s.srv.ListenAndServeTLS(apiConfig.TLS.CertFile, apiConfig.TLS.KeyFile); err != nil && err != http.ErrServerClosed {
				fmt.Printf("API Server TLS error: %v\n", err)
			}
		} else {
			if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("API Server error: %v\n", err)
			}
		}
	}()

	return nil
}

// Stop stops the API server.
func (s *Server) Stop(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}

func (s *Server) registerRoutes(r *mux.Router) {
	// API v1
	v1 := r.PathPrefix("/api/v1").Subrouter()

	// System
	r.HandleFunc("/health", s.handleHealth).Methods("GET")
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")
	r.HandleFunc("/api/v1/login", s.handleLogin).Methods("POST") // Public endpoint
	v1.HandleFunc("/status", s.handleStatus).Methods("GET")

	// Gateways
	v1.HandleFunc("/gateways", s.handleListGateways).Methods("GET")
	v1.HandleFunc("/gateways/{name}/send", s.handleSendGateway).Methods("POST")

	// Web Admin Dashboard (Serve static files)
	// Expects ./web/admin/dist to exist (run `npm run build` in web/admin)
	spa := http.StripPrefix("/admin/", http.FileServer(http.Dir("./web/admin/dist")))
	r.PathPrefix("/admin/").Handler(spa)
}
