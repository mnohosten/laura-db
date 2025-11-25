package server

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	// Create temporary directory for certificates
	tmpDir, err := os.MkdirTemp("", "tls-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	// Generate certificate
	err = GenerateSelfSignedCert(certFile, keyFile, "localhost")
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	// Check if files were created
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		t.Errorf("Certificate file was not created")
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		t.Errorf("Key file was not created")
	}

	// Try to load the certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Errorf("Failed to load generated certificate: %v", err)
	}

	// Parse the certificate
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Errorf("Failed to parse certificate: %v", err)
	}

	// Verify certificate properties
	if x509Cert.Subject.CommonName != "localhost" {
		t.Errorf("Expected CommonName 'localhost', got '%s'", x509Cert.Subject.CommonName)
	}

	// Check if certificate is valid
	now := time.Now()
	if now.Before(x509Cert.NotBefore) || now.After(x509Cert.NotAfter) {
		t.Errorf("Certificate is not currently valid")
	}

	// Check DNS names
	foundLocalhost := false
	for _, name := range x509Cert.DNSNames {
		if name == "localhost" || name == "127.0.0.1" {
			foundLocalhost = true
			break
		}
	}
	if !foundLocalhost {
		t.Errorf("Certificate does not include localhost or 127.0.0.1 in DNS names")
	}
}

func TestServerTLSConfiguration(t *testing.T) {
	// Create temporary directory for test data and certificates
	tmpDir, err := os.MkdirTemp("", "server-tls-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	// Generate self-signed certificate
	err = GenerateSelfSignedCert(certFile, keyFile, "localhost")
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	// Test 1: Server should fail if TLS is enabled but cert/key not specified
	config := DefaultConfig()
	config.DataDir = dataDir
	config.Port = 0 // Use random port
	config.EnableTLS = true
	config.TLSCertFile = ""
	config.TLSKeyFile = ""

	_, err = New(config)
	if err == nil {
		t.Error("Expected error when TLS enabled but cert/key not specified")
	}

	// Test 2: Server should fail if cert file doesn't exist
	config.TLSCertFile = filepath.Join(tmpDir, "nonexistent.pem")
	config.TLSKeyFile = keyFile

	_, err = New(config)
	if err == nil {
		t.Error("Expected error when cert file doesn't exist")
	}

	// Test 3: Server should fail if key file doesn't exist
	config.TLSCertFile = certFile
	config.TLSKeyFile = filepath.Join(tmpDir, "nonexistent.key")

	_, err = New(config)
	if err == nil {
		t.Error("Expected error when key file doesn't exist")
	}

	// Test 4: Server should start successfully with valid TLS configuration
	config.TLSCertFile = certFile
	config.TLSKeyFile = keyFile

	srv, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server with TLS: %v", err)
	}
	defer srv.Shutdown()

	// Verify TLS is enabled in config
	if !srv.config.EnableTLS {
		t.Error("TLS should be enabled")
	}
	if srv.config.TLSCertFile != certFile {
		t.Errorf("Expected cert file %s, got %s", certFile, srv.config.TLSCertFile)
	}
	if srv.config.TLSKeyFile != keyFile {
		t.Errorf("Expected key file %s, got %s", keyFile, srv.config.TLSKeyFile)
	}
}

func TestServerTLSConnection(t *testing.T) {
	// Create temporary directory for test data and certificates
	tmpDir, err := os.MkdirTemp("", "server-tls-conn-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	// Generate self-signed certificate
	err = GenerateSelfSignedCert(certFile, keyFile, "localhost")
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	// Create server with TLS
	config := DefaultConfig()
	config.DataDir = dataDir
	config.Host = "127.0.0.1"
	config.Port = 0 // Will be assigned automatically
	config.EnableTLS = true
	config.TLSCertFile = certFile
	config.TLSKeyFile = keyFile

	srv, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		err := srv.Start()
		if err != nil {
			errChan <- err
		}
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Get the actual port (if we used port 0)
	// Since we can't easily get the actual assigned port, we'll use a fixed port for this test
	config.Port = 18443 // Use non-standard HTTPS port
	srv.Shutdown()      // Shut down the first server

	// Recreate with fixed port
	srv, err = New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		err := srv.Start()
		if err != nil {
			errChan <- err
		}
	}()
	defer srv.Shutdown()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Create HTTPS client with custom transport that accepts self-signed certs
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Accept self-signed certificate
			},
		},
		Timeout: 5 * time.Second,
	}

	// Test HTTPS connection to health endpoint
	url := fmt.Sprintf("https://%s:%d/_health", config.Host, config.Port)
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("Failed to connect to HTTPS server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var healthResp map[string]interface{}
	if err := json.Unmarshal(body, &healthResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if ok, exists := healthResp["ok"].(bool); !exists || !ok {
		t.Errorf("Expected ok: true, got %v", healthResp["ok"])
	}

	if result, exists := healthResp["result"].(map[string]interface{}); exists {
		if status, ok := result["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", result["status"])
		}
	} else {
		t.Error("Expected result field in response")
	}
}

func TestServerHTTPConnection(t *testing.T) {
	// Create temporary directory for test data
	tmpDir, err := os.MkdirTemp("", "server-http-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")

	// Create server without TLS
	config := DefaultConfig()
	config.DataDir = dataDir
	config.Host = "127.0.0.1"
	config.Port = 18080 // Use non-standard HTTP port
	config.EnableTLS = false

	srv, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in background
	go func() {
		srv.Start()
	}()
	defer srv.Shutdown()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Test HTTP connection to health endpoint
	url := fmt.Sprintf("http://%s:%d/_health", config.Host, config.Port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to connect to HTTP server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var healthResp map[string]interface{}
	if err := json.Unmarshal(body, &healthResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if ok, exists := healthResp["ok"].(bool); !exists || !ok {
		t.Errorf("Expected ok: true, got %v", healthResp["ok"])
	}

	if result, exists := healthResp["result"].(map[string]interface{}); exists {
		if status, ok := result["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", result["status"])
		}
	} else {
		t.Error("Expected result field in response")
	}
}
