package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mnohosten/laura-db/pkg/server"
)

func main() {
	// Parse command-line flags
	host := flag.String("host", "localhost", "Server host address")
	port := flag.Int("port", 8080, "Server port")
	dataDir := flag.String("data-dir", "./data", "Data directory for database storage (persistent disk storage)")
	bufferSize := flag.Int("buffer-size", 1000, "Buffer pool size in pages (1 page = 4KB, default 1000 = ~4MB)")
	docCache := flag.Int("doc-cache", 1000, "Document cache size per collection (default: 1000 documents)")
	corsOrigin := flag.String("cors-origin", "*", "CORS allowed origin")
	enableTLS := flag.Bool("tls", false, "Enable TLS/SSL")
	tlsCert := flag.String("tls-cert", "", "Path to TLS certificate file")
	tlsKey := flag.String("tls-key", "", "Path to TLS private key file")
	enableGraphQL := flag.Bool("graphql", false, "Enable GraphQL API endpoint (/graphql) and GraphiQL playground (/graphiql)")
	flag.Parse()

	// Create server configuration
	config := server.DefaultConfig()
	config.Host = *host
	config.Port = *port
	config.DataDir = *dataDir
	config.BufferSize = *bufferSize
	config.DocumentCache = *docCache
	config.AllowedOrigins = []string{*corsOrigin}
	config.EnableTLS = *enableTLS
	config.TLSCertFile = *tlsCert
	config.TLSKeyFile = *tlsKey
	config.EnableGraphQL = *enableGraphQL

	// Create and start server
	srv, err := server.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create server: %v\n", err)
		os.Exit(1)
	}

	// Start server (blocks until shutdown)
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Server error: %v\n", err)
		os.Exit(1)
	}
}
