package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mnohosten/laura-db/pkg/database"
	gql "github.com/mnohosten/laura-db/pkg/graphql"
	"github.com/mnohosten/laura-db/pkg/metrics"
	"github.com/mnohosten/laura-db/pkg/server/handlers"
)

// Server represents the HTTP server for LauraDB
type Server struct {
	config               *Config
	db                   *database.Database
	router               *chi.Mux
	httpSrv              *http.Server
	startTime            time.Time
	metricsCollector     *metrics.MetricsCollector
	resourceTracker      *metrics.ResourceTracker
	promExporter         *metrics.PrometheusExporter
	changeStreamManager  *handlers.ChangeStreamManager
}

// New creates a new HTTP server instance
func New(config *Config) (*Server, error) {
	// Validate TLS configuration
	if config.EnableTLS {
		if config.TLSCertFile == "" || config.TLSKeyFile == "" {
			return nil, fmt.Errorf("TLS enabled but certificate or key file not specified")
		}
		// Check if certificate and key files exist
		if _, err := os.Stat(config.TLSCertFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("TLS certificate file not found: %s", config.TLSCertFile)
		}
		if _, err := os.Stat(config.TLSKeyFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("TLS key file not found: %s", config.TLSKeyFile)
		}
	}

	// Open database
	dbConfig := &database.Config{
		DataDir:        config.DataDir,
		BufferPoolSize: config.BufferSize,
	}
	db, err := database.Open(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create metrics collector and resource tracker
	metricsCollector := metrics.NewMetricsCollector()
	resourceTracker := metrics.NewResourceTracker(nil) // Use default config
	promExporter := metrics.NewPrometheusExporter(metricsCollector, resourceTracker)

	// Create server instance
	srv := &Server{
		config:          config,
		db:              db,
		router:          chi.NewRouter(),
		startTime:       time.Now(),
		metricsCollector: metricsCollector,
		resourceTracker:  resourceTracker,
		promExporter:     promExporter,
	}

	// Setup middleware
	srv.setupMiddleware()

	// Setup routes
	srv.setupRoutes()

	// Setup GraphQL routes if enabled
	if config.EnableGraphQL {
		if err := srv.setupGraphQLRoutes(); err != nil {
			return nil, fmt.Errorf("failed to setup GraphQL routes: %w", err)
		}
	}

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	srv.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      srv.router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	return srv, nil
}

// setupMiddleware configures HTTP middleware stack
func (s *Server) setupMiddleware() {
	// Request ID middleware
	s.router.Use(middleware.RequestID)

	// Real IP middleware
	s.router.Use(middleware.RealIP)

	// Recovery middleware to recover from panics
	s.router.Use(middleware.Recoverer)

	// Request logging
	if s.config.EnableLogging {
		s.router.Use(middleware.Logger)
	}

	// CORS middleware
	if s.config.EnableCORS {
		s.router.Use(s.corsMiddleware)
	}

	// Request size limit (but don't set Content-Type globally as it breaks static files)
	s.router.Use(s.requestSizeLimitMiddleware)

	// Timeout middleware
	s.router.Use(middleware.Timeout(60 * time.Second))
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes() {
	h := handlers.New(s.db)

	// Setup WebSocket routes for change streams
	manager, err := handlers.SetupWebSocketRoutes(s.router, h, s.config.DataDir)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to setup WebSocket routes: %v\n", err)
		fmt.Println("   WebSocket change streams will not be available")
	} else {
		s.changeStreamManager = manager
		fmt.Println("✅ WebSocket change streams enabled")
	}

	// Serve admin console static files
	workDir, _ := os.Getwd()
	filesDir := http.Dir(workDir + "/pkg/server/static")
	s.router.Handle("/static/*", http.StripPrefix("/static", http.FileServer(filesDir)))

	// Serve admin console root page
	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, workDir+"/pkg/server/static/index.html")
	})

	// Health and admin endpoints (API routes)
	s.router.Get("/_health", s.jsonContentType(h.Health(s.startTime)))
	s.router.Get("/_stats", s.jsonContentType(h.GetDatabaseStats))
	s.router.Get("/_collections", s.jsonContentType(h.ListCollections))

	// Prometheus metrics endpoint
	s.router.Get("/_metrics", s.handlePrometheusMetrics)

	// Cursor API endpoints
	s.router.Post("/_cursors", s.jsonContentType(h.CreateCursor))
	s.router.Get("/_cursors/{cursorId}/batch", s.jsonContentType(h.FetchBatch))
	s.router.Delete("/_cursors/{cursorId}", s.jsonContentType(h.CloseCursor))

	// Collection routes
	s.router.Route("/{collection}", func(r chi.Router) {
		// Set JSON content type for all collection routes
		r.Use(middleware.SetHeader("Content-Type", "application/json"))

		// Collection management
		r.Put("/", h.CreateCollection)
		r.Delete("/", h.DropCollection)
		r.Get("/_stats", h.GetCollectionStats)

		// Document operations
		r.Post("/_doc", h.InsertDocument)
		r.Post("/_doc/{id}", h.InsertDocumentWithID)
		r.Get("/_doc/{id}", h.GetDocument)
		r.Put("/_doc/{id}", h.UpdateDocument)
		r.Delete("/_doc/{id}", h.DeleteDocument)

		// Bulk operations
		r.Post("/_bulk", h.BulkInsert)
		r.Post("/_bulkWrite", h.BulkWrite)

		// Query operations
		r.Post("/_search", h.SearchDocuments)
		r.Get("/_count", h.CountDocuments)
		r.Post("/_count", h.CountDocumentsWithFilter)

		// Aggregation
		r.Post("/_aggregate", h.Aggregate)

		// Index management
		r.Post("/_index", h.CreateIndex)
		r.Get("/_index", h.ListIndexes)
		r.Delete("/_index/{name}", h.DropIndex)
	})
}

// setupGraphQLRoutes configures GraphQL routes
func (s *Server) setupGraphQLRoutes() error {
	// Create GraphQL handler
	graphqlHandler, err := gql.NewHandler(s.db)
	if err != nil {
		return fmt.Errorf("failed to create GraphQL handler: %w", err)
	}

	// Mount GraphQL endpoint
	s.router.Post("/graphql", graphqlHandler.ServeHTTP)

	// Mount GraphiQL playground (interactive UI)
	s.router.Get("/graphiql", gql.GraphiQLHandler())

	fmt.Println("✅ GraphQL API enabled")
	fmt.Printf("   GraphQL endpoint: /graphql\n")
	fmt.Printf("   GraphiQL playground: /graphiql\n")

	return nil
}

// jsonContentType middleware wraps a handler to set JSON content type
func (s *Server) jsonContentType(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next(w, r)
	}
}

// corsMiddleware handles CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		origin := "*"
		if len(s.config.AllowedOrigins) > 0 {
			origin = s.config.AllowedOrigins[0]
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requestSizeLimitMiddleware limits request body size
func (s *Server) requestSizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxRequestSize)
		next.ServeHTTP(w, r)
	})
}

// handlePrometheusMetrics handles the Prometheus metrics endpoint
func (s *Server) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	// Set Prometheus text format content type
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Write metrics
	if err := s.promExporter.WriteMetrics(w); err != nil {
		http.Error(w, fmt.Sprintf("Error writing metrics: %v", err), http.StatusInternalServerError)
		return
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	protocol := "http"
	wsProtocol := "ws"
	if s.config.EnableTLS {
		protocol = "https"
		wsProtocol = "wss"
		fmt.Printf("🔒 TLS/SSL enabled\n")
		fmt.Printf("📜 Certificate: %s\n", s.config.TLSCertFile)
	}
	fmt.Printf("🚀 LauraDB server starting on %s://%s:%d\n", protocol, s.config.Host, s.config.Port)
	fmt.Printf("📁 Data directory: %s\n", s.config.DataDir)
	fmt.Printf("💾 Buffer pool size: %d pages\n", s.config.BufferSize)
	if s.changeStreamManager != nil {
		fmt.Printf("🔌 WebSocket endpoint: %s://%s:%d/_ws/watch\n", wsProtocol, s.config.Host, s.config.Port)
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		var err error
		if s.config.EnableTLS {
			err = s.httpSrv.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
		} else {
			err = s.httpSrv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for either error or shutdown signal
	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan:
		fmt.Printf("\n⚠️  Received signal: %v\n", sig)
		return s.Shutdown()
	}
}

// GetDatabase returns the database instance
func (s *Server) GetDatabase() *database.Database {
	return s.db
}

// GetMetricsCollector returns the metrics collector
func (s *Server) GetMetricsCollector() *metrics.MetricsCollector {
	return s.metricsCollector
}

// GetResourceTracker returns the resource tracker
func (s *Server) GetResourceTracker() *metrics.ResourceTracker {
	return s.resourceTracker
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	fmt.Println("🛑 Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		fmt.Printf("❌ Server shutdown error: %v\n", err)
	}

	// Close change stream manager and all active WebSocket connections
	if s.changeStreamManager != nil {
		if err := s.changeStreamManager.Close(); err != nil {
			fmt.Printf("⚠️  Warning: Error closing change stream manager: %v\n", err)
		}
	}

	// Stop resource tracker
	if s.resourceTracker != nil {
		s.resourceTracker.Disable()
	}

	// Close database
	if err := s.db.Close(); err != nil {
		fmt.Printf("❌ Database close error: %v\n", err)
		return err
	}

	fmt.Println("✅ Server shutdown complete")
	return nil
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("Error encoding JSON response: %v\n", err)
	}
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, statusCode int, errorType, message string) {
	response := map[string]interface{}{
		"ok":      false,
		"error":   errorType,
		"message": message,
		"code":    statusCode,
	}
	WriteJSON(w, statusCode, response)
}

// WriteSuccess writes a success response
func WriteSuccess(w http.ResponseWriter, result interface{}) {
	response := map[string]interface{}{
		"ok":     true,
		"result": result,
	}
	WriteJSON(w, http.StatusOK, response)
}

// WriteSuccessWithCount writes a success response with count
func WriteSuccessWithCount(w http.ResponseWriter, result interface{}, count int) {
	response := map[string]interface{}{
		"ok":     true,
		"result": result,
		"count":  count,
	}
	WriteJSON(w, http.StatusOK, response)
}
