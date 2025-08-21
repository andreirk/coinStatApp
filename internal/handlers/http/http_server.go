package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"coinStatApp/internal/domain/service"
	"coinStatApp/internal/handlers/websocket"
)

// Server represents an HTTP server with all routes configured
type Server struct {
	statsService *service.TimeWindowedStatisticsService
	broadcaster  *websocket.WebSocketBroadcaster
	mux          *http.ServeMux
	server       *http.Server
}

// NewServer creates a new HTTP server with configured routes
func NewServer(addr string, statsService *service.TimeWindowedStatisticsService, broadcaster *websocket.WebSocketBroadcaster) *Server {
	mux := http.NewServeMux()

	server := &Server{
		statsService: statsService,
		broadcaster:  broadcaster,
		mux:          mux,
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

	// Register routes
	server.registerRoutes()

	return server
}

// registerRoutes configures all HTTP routes
func (s *Server) registerRoutes() {
	// Statistics endpoint
	s.mux.HandleFunc("/stats", s.handleStats)

	// Health check endpoint
	s.mux.HandleFunc("/health", s.handleHealth)

	// WebSocket endpoint
	s.mux.HandleFunc("/ws", s.broadcaster.Handler())
}

// handleStats handles requests for statistics data
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.statsService.GetAllStatistics(r.Context())
	if err != nil {
		http.Error(w, "failed to get stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "app/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("failed to encode stats: %v", err)
	}
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "app/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Start begins listening for HTTP requests
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
