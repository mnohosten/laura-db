package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mnohosten/laura-db/pkg/server"
)

func main() {
	fmt.Println("LauraDB TLS/SSL Demo")
	fmt.Println("====================")
	fmt.Println()

	// Create temporary directory for demo
	tmpDir, err := os.MkdirTemp("", "laura-tls-demo-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	fmt.Println("Demo 1: Generate Self-Signed Certificate")
	fmt.Println("-----------------------------------------")
	fmt.Printf("Generating certificate for localhost...\n")

	err = server.GenerateSelfSignedCert(certFile, keyFile, "localhost")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate certificate: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Certificate generated: %s\n", certFile)
	fmt.Printf("✓ Private key generated: %s\n", keyFile)
	fmt.Println()

	fmt.Println("Demo 2: Start HTTPS Server")
	fmt.Println("--------------------------")

	// Create server configuration with TLS enabled
	config := server.DefaultConfig()
	config.DataDir = dataDir
	config.Host = "localhost"
	config.Port = 8443 // Standard HTTPS port (or use 8443 for development)
	config.EnableTLS = true
	config.TLSCertFile = certFile
	config.TLSKeyFile = keyFile
	config.EnableLogging = false // Disable request logging for cleaner demo output

	// Create and start server
	srv, err := server.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}

	// Start server in background
	go func() {
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)
	fmt.Println()

	fmt.Println("Demo 3: Connect to HTTPS Server")
	fmt.Println("-------------------------------")

	// Create HTTPS client that accepts self-signed certificates
	// In production, you should verify certificates properly!
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Accept self-signed certificate (development only!)
			},
		},
		Timeout: 5 * time.Second,
	}

	// Test connection to health endpoint
	url := fmt.Sprintf("https://%s:%d/_health", config.Host, config.Port)
	fmt.Printf("Making HTTPS request to: %s\n", url)

	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		srv.Shutdown()
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read response: %v\n", err)
		srv.Shutdown()
		os.Exit(1)
	}

	var healthResp map[string]interface{}
	if err := json.Unmarshal(body, &healthResp); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse response: %v\n", err)
		srv.Shutdown()
		os.Exit(1)
	}

	fmt.Printf("✓ HTTPS connection successful!\n")
	fmt.Printf("  Status: %d\n", resp.StatusCode)
	fmt.Printf("  TLS Version: %s\n", tlsVersionToString(resp.TLS.Version))
	fmt.Printf("  Cipher Suite: %s\n", tls.CipherSuiteName(resp.TLS.CipherSuite))
	fmt.Printf("  Server Response: %s\n", string(body))
	fmt.Println()

	fmt.Println("Demo 4: Verify Certificate Properties")
	fmt.Println("-------------------------------------")

	// Load and parse the certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load certificate: %v\n", err)
		srv.Shutdown()
		os.Exit(1)
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse certificate: %v\n", err)
		srv.Shutdown()
		os.Exit(1)
	}

	fmt.Printf("Certificate Information:\n")
	fmt.Printf("  Subject: %s\n", x509Cert.Subject.CommonName)
	fmt.Printf("  Issuer: %s\n", x509Cert.Issuer.CommonName)
	fmt.Printf("  Valid From: %s\n", x509Cert.NotBefore.Format(time.RFC3339))
	fmt.Printf("  Valid Until: %s\n", x509Cert.NotAfter.Format(time.RFC3339))
	fmt.Printf("  DNS Names: %v\n", x509Cert.DNSNames)
	fmt.Printf("  Serial Number: %s\n", x509Cert.SerialNumber.String())
	fmt.Println()

	fmt.Println("Demo 5: Security Comparison")
	fmt.Println("---------------------------")
	fmt.Println("HTTP (without TLS):")
	fmt.Println("  ✗ Data transmitted in plain text")
	fmt.Println("  ✗ Vulnerable to eavesdropping")
	fmt.Println("  ✗ No server authentication")
	fmt.Println("  ✗ Susceptible to man-in-the-middle attacks")
	fmt.Println()
	fmt.Println("HTTPS (with TLS):")
	fmt.Println("  ✓ Data encrypted in transit")
	fmt.Println("  ✓ Protected against eavesdropping")
	fmt.Println("  ✓ Server authentication via certificates")
	fmt.Println("  ✓ Protection against tampering")
	fmt.Println()

	fmt.Println("Demo Complete!")
	fmt.Println("=============")
	fmt.Println()
	fmt.Println("The server is running with TLS enabled.")
	fmt.Printf("You can access it at: https://%s:%d/\n", config.Host, config.Port)
	fmt.Println()
	fmt.Println("Note: For production use:")
	fmt.Println("  1. Use certificates from a trusted Certificate Authority (CA)")
	fmt.Println("  2. Never use InsecureSkipVerify in production clients")
	fmt.Println("  3. Keep private keys secure and never commit them to version control")
	fmt.Println("  4. Regularly renew certificates before expiration")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop the server...")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	srv.Shutdown()
	fmt.Println("Demo finished!")
}

func tlsVersionToString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}
