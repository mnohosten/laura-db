package server

import "time"

// Config holds server configuration settings
type Config struct {
	Host           string        // Server host address
	Port           int           // Server port
	DataDir        string        // Database data directory - where all database files are stored
	BufferSize     int           // Buffer pool size in pages (1 page = 4KB). Default: 1000 pages (~4MB)
	DocumentCache  int           // Per-collection document cache size. Default: 1000 documents
	ReadTimeout    time.Duration // HTTP read timeout
	WriteTimeout   time.Duration // HTTP write timeout
	IdleTimeout    time.Duration // HTTP idle timeout
	MaxRequestSize int64         // Maximum request body size in bytes
	EnableCORS     bool          // Enable CORS middleware
	AllowedOrigins []string      // CORS allowed origins
	AllowedMethods []string      // CORS allowed methods
	AllowedHeaders []string      // CORS allowed headers
	EnableLogging  bool          // Enable request logging
	LogFormat      string        // Log format (text or json)

	// TLS/SSL configuration
	EnableTLS   bool   // Enable TLS/SSL
	TLSCertFile string // Path to TLS certificate file
	TLSKeyFile  string // Path to TLS private key file

	// GraphQL configuration
	EnableGraphQL bool // Enable GraphQL API endpoint
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Host:           "localhost",
		Port:           8080,
		DataDir:        "./data",
		BufferSize:     1000,        // 1000 pages = ~4MB buffer pool
		DocumentCache:  1000,        // 1000 documents per collection cache
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxRequestSize: 10 * 1024 * 1024, // 10MB
		EnableCORS:     true,
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Request-ID"},
		EnableLogging:  true,
		LogFormat:      "text",
		EnableTLS:      false, // TLS disabled by default
		TLSCertFile:    "",
		TLSKeyFile:     "",
		EnableGraphQL:  false, // GraphQL disabled by default (opt-in feature)
	}
}
