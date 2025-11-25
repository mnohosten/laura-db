package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client represents a LauraDB client connection
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Config holds configuration for the client
type Config struct {
	// Host is the server hostname or IP address (default: "localhost")
	Host string
	// Port is the server port (default: 8080)
	Port int
	// Timeout is the HTTP request timeout (default: 30s)
	Timeout time.Duration
	// MaxIdleConns is the maximum number of idle connections (default: 10)
	MaxIdleConns int
	// MaxConnsPerHost is the maximum connections per host (default: 10)
	MaxConnsPerHost int
}

// DefaultConfig returns the default client configuration
func DefaultConfig() *Config {
	return &Config{
		Host:            "localhost",
		Port:            8080,
		Timeout:         30 * time.Second,
		MaxIdleConns:    10,
		MaxConnsPerHost: 10,
	}
}

// NewClient creates a new LauraDB client with the given configuration
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	// Apply defaults for unset fields
	if config.Host == "" {
		config.Host = "localhost"
	}
	if config.Port == 0 {
		config.Port = 8080
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 10
	}
	if config.MaxConnsPerHost == 0 {
		config.MaxConnsPerHost = 10
	}

	// Create HTTP client with custom transport
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxConnsPerHost:     config.MaxConnsPerHost,
		MaxIdleConnsPerHost: config.MaxConnsPerHost,
		IdleConnTimeout:     90 * time.Second,
	}

	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.Host, config.Port)

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// NewDefaultClient creates a client with default configuration
func NewDefaultClient() *Client {
	return NewClient(DefaultConfig())
}

// Response represents a standard API response
type Response struct {
	OK      bool            `json:"ok"`
	Result  json.RawMessage `json:"result,omitempty"`
	Count   *int            `json:"count,omitempty"`
	Error   string          `json:"error,omitempty"`
	Message string          `json:"message,omitempty"`
	Code    int             `json:"code,omitempty"`
}

// doRequest performs an HTTP request and returns the response
func (c *Client) doRequest(method, path string, body interface{}) (*Response, error) {
	// Build URL
	reqURL := c.baseURL + path

	// Encode request body if provided
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	// Create HTTP request
	req, err := http.NewRequest(method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var apiResp Response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API-level errors
	if !apiResp.OK {
		return &apiResp, fmt.Errorf("API error: %s - %s", apiResp.Error, apiResp.Message)
	}

	return &apiResp, nil
}

// Health checks the server health
func (c *Client) Health() (*HealthResponse, error) {
	resp, err := c.doRequest("GET", "/_health", nil)
	if err != nil {
		return nil, err
	}

	var health HealthResponse
	if err := json.Unmarshal(resp.Result, &health); err != nil {
		return nil, fmt.Errorf("failed to parse health response: %w", err)
	}

	return &health, nil
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string    `json:"status"`
	Uptime string    `json:"uptime"`
	Time   time.Time `json:"time"`
}

// Stats retrieves database statistics
func (c *Client) Stats() (*DatabaseStats, error) {
	resp, err := c.doRequest("GET", "/_stats", nil)
	if err != nil {
		return nil, err
	}

	var stats DatabaseStats
	if err := json.Unmarshal(resp.Result, &stats); err != nil {
		return nil, fmt.Errorf("failed to parse stats response: %w", err)
	}

	return &stats, nil
}

// DatabaseStats represents database statistics
type DatabaseStats struct {
	Name               string                       `json:"name"`
	Collections        int                          `json:"collections"`
	ActiveTransactions int                          `json:"active_transactions"`
	CollectionStats    map[string]CollectionStats   `json:"collection_stats"`
	StorageStats       StorageStats                 `json:"storage_stats"`
}

// CollectionStats represents statistics for a single collection
type CollectionStats struct {
	Name         string      `json:"name"`
	Count        int         `json:"count"`
	Indexes      int         `json:"indexes"`
	IndexDetails []IndexInfo `json:"index_details"`
}

// IndexInfo represents information about an index
type IndexInfo struct {
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Fields map[string]int         `json:"fields"`
	Unique bool                   `json:"unique"`
	Sparse bool                   `json:"sparse"`
	Stats  map[string]interface{} `json:"stats,omitempty"`
}

// StorageStats represents storage-level statistics
type StorageStats struct {
	BufferPool BufferPoolStats `json:"buffer_pool"`
	Disk       DiskStats       `json:"disk"`
}

// BufferPoolStats represents buffer pool statistics
type BufferPoolStats struct {
	Capacity  int     `json:"capacity"`
	Size      int     `json:"size"`
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	HitRate   float64 `json:"hit_rate"`
	Evictions int64   `json:"evictions"`
}

// DiskStats represents disk-level statistics
type DiskStats struct {
	TotalReads  int64 `json:"total_reads"`
	TotalWrites int64 `json:"total_writes"`
	NextPageID  int   `json:"next_page_id"`
	FreePages   int   `json:"free_pages"`
}

// ListCollections returns a list of all collections
func (c *Client) ListCollections() ([]string, error) {
	resp, err := c.doRequest("GET", "/_collections", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Collections []string `json:"collections"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse collections response: %w", err)
	}

	return result.Collections, nil
}

// Collection returns a collection handle for the given name
func (c *Client) Collection(name string) *Collection {
	return &Collection{
		client: c,
		name:   name,
	}
}

// CreateCollection creates a new collection
func (c *Client) CreateCollection(name string) error {
	path := "/" + url.PathEscape(name)
	_, err := c.doRequest("PUT", path, nil)
	return err
}

// DropCollection drops a collection
func (c *Client) DropCollection(name string) error {
	path := "/" + url.PathEscape(name)
	_, err := c.doRequest("DELETE", path, nil)
	return err
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	// Close idle connections
	c.httpClient.CloseIdleConnections()
	return nil
}
