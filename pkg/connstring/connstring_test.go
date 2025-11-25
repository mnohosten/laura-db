package connstring

import (
	"testing"
	"time"
)

func TestParse_BasicLaura(t *testing.T) {
	cs, err := Parse("laura://localhost:8080")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Scheme != "laura" {
		t.Errorf("expected scheme 'laura', got '%s'", cs.Scheme)
	}

	if len(cs.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(cs.Hosts))
	}

	if cs.Hosts[0].Host != "localhost" || cs.Hosts[0].Port != 8080 {
		t.Errorf("expected localhost:8080, got %s:%d", cs.Hosts[0].Host, cs.Hosts[0].Port)
	}
}

func TestParse_BasicMongoDB(t *testing.T) {
	cs, err := Parse("mongodb://localhost:27017")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Scheme != "mongodb" {
		t.Errorf("expected scheme 'mongodb', got '%s'", cs.Scheme)
	}

	if cs.Hosts[0].Port != 27017 {
		t.Errorf("expected port 27017, got %d", cs.Hosts[0].Port)
	}
}

func TestParse_WithDatabase(t *testing.T) {
	cs, err := Parse("laura://localhost:8080/mydb")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Database != "mydb" {
		t.Errorf("expected database 'mydb', got '%s'", cs.Database)
	}
}

func TestParse_WithAuthentication(t *testing.T) {
	cs, err := Parse("mongodb://user:pass@localhost:27017/mydb")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Options.Username != "user" {
		t.Errorf("expected username 'user', got '%s'", cs.Options.Username)
	}

	if cs.Options.Password != "pass" {
		t.Errorf("expected password 'pass', got '%s'", cs.Options.Password)
	}
}

func TestParse_WithEncodedPassword(t *testing.T) {
	cs, err := Parse("mongodb://user:p%40ss%21@localhost:27017/mydb")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Options.Username != "user" {
		t.Errorf("expected username 'user', got '%s'", cs.Options.Username)
	}

	if cs.Options.Password != "p@ss!" {
		t.Errorf("expected password 'p@ss!', got '%s'", cs.Options.Password)
	}
}

func TestParse_MultipleHosts(t *testing.T) {
	cs, err := Parse("laura://host1:8080,host2:8081,host3:8082/mydb")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(cs.Hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d", len(cs.Hosts))
	}

	expected := []Host{
		{Host: "host1", Port: 8080},
		{Host: "host2", Port: 8081},
		{Host: "host3", Port: 8082},
	}

	for i, exp := range expected {
		if cs.Hosts[i].Host != exp.Host || cs.Hosts[i].Port != exp.Port {
			t.Errorf("host %d: expected %s:%d, got %s:%d",
				i, exp.Host, exp.Port, cs.Hosts[i].Host, cs.Hosts[i].Port)
		}
	}
}

func TestParse_DefaultPort(t *testing.T) {
	cs, err := Parse("laura://localhost")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Hosts[0].Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cs.Hosts[0].Port)
	}
}

func TestParse_WithOptions(t *testing.T) {
	cs, err := Parse("laura://localhost:8080/mydb?timeout=5000&maxConnections=50&tls=true&appName=myapp")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Options.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", cs.Options.Timeout)
	}

	if cs.Options.MaxConnections != 50 {
		t.Errorf("expected maxConnections 50, got %d", cs.Options.MaxConnections)
	}

	if !cs.Options.TLS {
		t.Error("expected TLS to be enabled")
	}

	if cs.Options.AppName != "myapp" {
		t.Errorf("expected appName 'myapp', got '%s'", cs.Options.AppName)
	}
}

func TestParse_ReadWriteOptions(t *testing.T) {
	cs, err := Parse("laura://localhost?readPreference=secondary&writeConcern=1")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Options.ReadPreference != "secondary" {
		t.Errorf("expected readPreference 'secondary', got '%s'", cs.Options.ReadPreference)
	}

	if cs.Options.WriteConcern != "1" {
		t.Errorf("expected writeConcern '1', got '%s'", cs.Options.WriteConcern)
	}
}

func TestParse_TLSOptions(t *testing.T) {
	cs, err := Parse("laura://localhost?tls=true&tlsInsecure=true&tlsCertFile=/path/cert.pem&tlsKeyFile=/path/key.pem")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !cs.Options.TLS {
		t.Error("expected TLS to be enabled")
	}

	if !cs.Options.TLSInsecure {
		t.Error("expected TLSInsecure to be enabled")
	}

	if cs.Options.TLSCertFile != "/path/cert.pem" {
		t.Errorf("expected tlsCertFile '/path/cert.pem', got '%s'", cs.Options.TLSCertFile)
	}

	if cs.Options.TLSKeyFile != "/path/key.pem" {
		t.Errorf("expected tlsKeyFile '/path/key.pem', got '%s'", cs.Options.TLSKeyFile)
	}
}

func TestParse_ReplicaSetOptions(t *testing.T) {
	cs, err := Parse("mongodb://host1,host2,host3/mydb?replicaSet=rs0&retryWrites=false")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Options.ReplicaSet != "rs0" {
		t.Errorf("expected replicaSet 'rs0', got '%s'", cs.Options.ReplicaSet)
	}

	if cs.Options.RetryWrites {
		t.Error("expected retryWrites to be disabled")
	}
}

func TestParse_AuthOptions(t *testing.T) {
	cs, err := Parse("mongodb://user:pass@localhost/mydb?authSource=admin&authMechanism=SCRAM-SHA-256")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Options.AuthDB != "admin" {
		t.Errorf("expected authSource 'admin', got '%s'", cs.Options.AuthDB)
	}

	if cs.Options.AuthMech != "SCRAM-SHA-256" {
		t.Errorf("expected authMechanism 'SCRAM-SHA-256', got '%s'", cs.Options.AuthMech)
	}
}

func TestParse_BooleanValues(t *testing.T) {
	tests := []struct {
		uri      string
		expected bool
	}{
		{"laura://localhost?tls=true", true},
		{"laura://localhost?tls=false", false},
		{"laura://localhost?tls=1", true},
		{"laura://localhost?tls=0", false},
		{"laura://localhost?tls=yes", true},
		{"laura://localhost?tls=no", false},
		{"laura://localhost?tls=TRUE", true},
		{"laura://localhost?tls=FALSE", false},
	}

	for _, tt := range tests {
		cs, err := Parse(tt.uri)
		if err != nil {
			t.Fatalf("Parse failed for %s: %v", tt.uri, err)
		}

		if cs.Options.TLS != tt.expected {
			t.Errorf("for %s: expected TLS=%v, got %v", tt.uri, tt.expected, cs.Options.TLS)
		}
	}
}

func TestParse_InvalidScheme(t *testing.T) {
	_, err := Parse("http://localhost:8080")
	if err != ErrInvalidScheme {
		t.Errorf("expected ErrInvalidScheme, got %v", err)
	}
}

func TestParse_EmptyString(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Error("expected error for empty connection string")
	}
}

func TestParse_NoHosts(t *testing.T) {
	_, err := Parse("laura:///mydb")
	if err != ErrNoHosts {
		t.Errorf("expected ErrNoHosts, got %v", err)
	}
}

func TestParse_InvalidPort(t *testing.T) {
	_, err := Parse("laura://localhost:invalid")
	if err == nil {
		t.Error("expected error for invalid port")
	}
}

func TestParse_PortOutOfRange(t *testing.T) {
	_, err := Parse("laura://localhost:99999")
	if err == nil {
		t.Error("expected error for port out of range")
	}
}

func TestParse_InvalidTimeout(t *testing.T) {
	_, err := Parse("laura://localhost?timeout=invalid")
	if err == nil {
		t.Error("expected error for invalid timeout")
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "laura://localhost:8080",
			expected: "laura://localhost:8080",
		},
		{
			input:    "laura://localhost",
			expected: "laura://localhost:8080", // default port added
		},
		{
			input:    "mongodb://localhost:27017/mydb",
			expected: "mongodb://localhost:27017/mydb",
		},
		{
			input:    "laura://user:pass@localhost:8080/mydb",
			expected: "laura://user:pass@localhost:8080/mydb",
		},
		{
			input:    "laura://host1:8080,host2:8081,host3:8082",
			expected: "laura://host1:8080,host2:8081,host3:8082",
		},
	}

	for _, tt := range tests {
		cs, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse failed for %s: %v", tt.input, err)
		}

		result := cs.String()
		if result != tt.expected {
			t.Errorf("String() for %s: expected '%s', got '%s'", tt.input, tt.expected, result)
		}
	}
}

func TestGetFirstHost(t *testing.T) {
	cs, _ := Parse("laura://host1:8080,host2:8081")
	host := cs.GetFirstHost()

	if host.Host != "host1" || host.Port != 8080 {
		t.Errorf("expected host1:8080, got %s:%d", host.Host, host.Port)
	}
}

func TestGetFirstHost_Empty(t *testing.T) {
	cs := &ConnString{}
	host := cs.GetFirstHost()

	if host.Host != "localhost" || host.Port != 8080 {
		t.Errorf("expected default localhost:8080, got %s:%d", host.Host, host.Port)
	}
}

func TestHasAuthentication(t *testing.T) {
	cs1, _ := Parse("laura://user:pass@localhost")
	if !cs1.HasAuthentication() {
		t.Error("expected HasAuthentication to be true")
	}

	cs2, _ := Parse("laura://localhost")
	if cs2.HasAuthentication() {
		t.Error("expected HasAuthentication to be false")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", opts.Timeout)
	}

	if opts.MaxConnections != 100 {
		t.Errorf("expected default maxConnections 100, got %d", opts.MaxConnections)
	}

	if opts.ReadPreference != "primary" {
		t.Errorf("expected default readPreference 'primary', got '%s'", opts.ReadPreference)
	}

	if !opts.RetryWrites {
		t.Error("expected default retryWrites to be true")
	}
}

func TestParse_ComplexRealWorld(t *testing.T) {
	uri := "mongodb://admin:P%40ssw0rd%21@host1.example.com:27017,host2.example.com:27017,host3.example.com:27017/production?replicaSet=rs0&readPreference=secondaryPreferred&maxConnections=200&timeout=10000&tls=true&appName=MyApp&retryWrites=true"

	cs, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify scheme
	if cs.Scheme != "mongodb" {
		t.Errorf("expected scheme 'mongodb', got '%s'", cs.Scheme)
	}

	// Verify authentication
	if cs.Options.Username != "admin" {
		t.Errorf("expected username 'admin', got '%s'", cs.Options.Username)
	}
	if cs.Options.Password != "P@ssw0rd!" {
		t.Errorf("expected decoded password 'P@ssw0rd!', got '%s'", cs.Options.Password)
	}

	// Verify hosts
	if len(cs.Hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d", len(cs.Hosts))
	}

	// Verify database
	if cs.Database != "production" {
		t.Errorf("expected database 'production', got '%s'", cs.Database)
	}

	// Verify options
	if cs.Options.ReplicaSet != "rs0" {
		t.Errorf("expected replicaSet 'rs0', got '%s'", cs.Options.ReplicaSet)
	}
	if cs.Options.ReadPreference != "secondaryPreferred" {
		t.Errorf("expected readPreference 'secondaryPreferred', got '%s'", cs.Options.ReadPreference)
	}
	if cs.Options.MaxConnections != 200 {
		t.Errorf("expected maxConnections 200, got %d", cs.Options.MaxConnections)
	}
	if cs.Options.Timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", cs.Options.Timeout)
	}
	if !cs.Options.TLS {
		t.Error("expected TLS to be enabled")
	}
	if cs.Options.AppName != "MyApp" {
		t.Errorf("expected appName 'MyApp', got '%s'", cs.Options.AppName)
	}
	if !cs.Options.RetryWrites {
		t.Error("expected retryWrites to be enabled")
	}
}

// TestParse_AllConnectionOptions tests all connection-related options
func TestParse_AllConnectionOptions(t *testing.T) {
	uri := "laura://localhost?minConnections=5&maxConnections=100&maxIdleTime=30000&connectTimeout=5000&socketTimeout=60000"
	cs, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Options.MinConnections != 5 {
		t.Errorf("expected minConnections 5, got %d", cs.Options.MinConnections)
	}
	if cs.Options.MaxConnections != 100 {
		t.Errorf("expected maxConnections 100, got %d", cs.Options.MaxConnections)
	}
	if cs.Options.MaxIdleTime != 30*time.Second {
		t.Errorf("expected maxIdleTime 30s, got %v", cs.Options.MaxIdleTime)
	}
	if cs.Options.ConnectTimeout != 5*time.Second {
		t.Errorf("expected connectTimeout 5s, got %v", cs.Options.ConnectTimeout)
	}
	if cs.Options.SocketTimeout != 60*time.Second {
		t.Errorf("expected socketTimeout 60s, got %v", cs.Options.SocketTimeout)
	}
}

// TestParse_AllTLSOptions tests all TLS-related options
func TestParse_AllTLSOptions(t *testing.T) {
	uri := "laura://localhost?tls=true&tlsInsecure=false&tlsCertFile=/cert.pem&tlsKeyFile=/key.pem&tlsCAFile=/ca.pem"
	cs, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !cs.Options.TLS {
		t.Error("expected TLS to be enabled")
	}
	if cs.Options.TLSInsecure {
		t.Error("expected TLSInsecure to be false")
	}
	if cs.Options.TLSCertFile != "/cert.pem" {
		t.Errorf("expected tlsCertFile '/cert.pem', got '%s'", cs.Options.TLSCertFile)
	}
	if cs.Options.TLSKeyFile != "/key.pem" {
		t.Errorf("expected tlsKeyFile '/key.pem', got '%s'", cs.Options.TLSKeyFile)
	}
	if cs.Options.TLSCAFile != "/ca.pem" {
		t.Errorf("expected tlsCAFile '/ca.pem', got '%s'", cs.Options.TLSCAFile)
	}
}

// TestParse_RetryOptions tests retry-related options
func TestParse_RetryOptions(t *testing.T) {
	uri := "laura://localhost?retryWrites=true&retryReads=true"
	cs, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !cs.Options.RetryWrites {
		t.Error("expected retryWrites to be enabled")
	}
	if !cs.Options.RetryReads {
		t.Error("expected retryReads to be enabled")
	}
}

// TestParse_RetryOptionsDisabled tests retry options when disabled
func TestParse_RetryOptionsDisabled(t *testing.T) {
	uri := "laura://localhost?retryWrites=false&retryReads=false"
	cs, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Options.RetryWrites {
		t.Error("expected retryWrites to be disabled")
	}
	if cs.Options.RetryReads {
		t.Error("expected retryReads to be disabled")
	}
}

// TestParse_InvalidMinConnections tests error handling for invalid minConnections
func TestParse_InvalidMinConnections(t *testing.T) {
	uri := "laura://localhost?minConnections=invalid"
	_, err := Parse(uri)
	if err == nil {
		t.Error("expected error for invalid minConnections")
	}
}

// TestParse_InvalidMaxIdleTime tests error handling for invalid maxIdleTime
func TestParse_InvalidMaxIdleTime(t *testing.T) {
	uri := "laura://localhost?maxIdleTime=invalid"
	_, err := Parse(uri)
	if err == nil {
		t.Error("expected error for invalid maxIdleTime")
	}
}

// TestParse_InvalidConnectTimeout tests error handling for invalid connectTimeout
func TestParse_InvalidConnectTimeout(t *testing.T) {
	uri := "laura://localhost?connectTimeout=invalid"
	_, err := Parse(uri)
	if err == nil {
		t.Error("expected error for invalid connectTimeout")
	}
}

// TestParse_InvalidSocketTimeout tests error handling for invalid socketTimeout
func TestParse_InvalidSocketTimeout(t *testing.T) {
	uri := "laura://localhost?socketTimeout=invalid"
	_, err := Parse(uri)
	if err == nil {
		t.Error("expected error for invalid socketTimeout")
	}
}

// TestParse_AlternativeOptionNames tests alternative option names (MongoDB compatibility)
func TestParse_AlternativeOptionNames(t *testing.T) {
	uri := "mongodb://localhost?minPoolSize=10&maxPoolSize=50&maxIdleTimeMS=15000&connectTimeoutMS=3000&socketTimeoutMS=45000"
	cs, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Options.MinConnections != 10 {
		t.Errorf("expected minConnections 10, got %d", cs.Options.MinConnections)
	}
	if cs.Options.MaxConnections != 50 {
		t.Errorf("expected maxConnections 50, got %d", cs.Options.MaxConnections)
	}
	if cs.Options.MaxIdleTime != 15*time.Second {
		t.Errorf("expected maxIdleTime 15s, got %v", cs.Options.MaxIdleTime)
	}
	if cs.Options.ConnectTimeout != 3*time.Second {
		t.Errorf("expected connectTimeout 3s, got %v", cs.Options.ConnectTimeout)
	}
	if cs.Options.SocketTimeout != 45*time.Second {
		t.Errorf("expected socketTimeout 45s, got %v", cs.Options.SocketTimeout)
	}
}

// TestParse_MiscellaneousOptions tests appName and other options
func TestParse_MiscellaneousOptions(t *testing.T) {
	uri := "laura://localhost?appName=TestApp&readPreference=secondary&writeConcern=majority&authSource=admin&authMechanism=SCRAM-SHA-256"
	cs, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Options.AppName != "TestApp" {
		t.Errorf("expected appName 'TestApp', got '%s'", cs.Options.AppName)
	}
	if cs.Options.ReadPreference != "secondary" {
		t.Errorf("expected readPreference 'secondary', got '%s'", cs.Options.ReadPreference)
	}
	if cs.Options.WriteConcern != "majority" {
		t.Errorf("expected writeConcern 'majority', got '%s'", cs.Options.WriteConcern)
	}
	if cs.Options.AuthDB != "admin" {
		t.Errorf("expected authSource 'admin', got '%s'", cs.Options.AuthDB)
	}
	if cs.Options.AuthMech != "SCRAM-SHA-256" {
		t.Errorf("expected authMechanism 'SCRAM-SHA-256', got '%s'", cs.Options.AuthMech)
	}
}

// TestParse_SSLAliasForTLS tests that 'ssl' works as an alias for 'tls'
func TestParse_SSLAliasForTLS(t *testing.T) {
	uri := "laura://localhost?ssl=true&tlsInsecureSkipVerify=true"
	cs, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !cs.Options.TLS {
		t.Error("expected TLS to be enabled via 'ssl' alias")
	}
	if !cs.Options.TLSInsecure {
		t.Error("expected TLSInsecure to be enabled via 'tlsInsecureSkipVerify' alias")
	}
}

// TestParse_WriteConcernAlias tests 'w' as alias for 'writeConcern'
func TestParse_WriteConcernAlias(t *testing.T) {
	uri := "laura://localhost?w=2"
	cs, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Options.WriteConcern != "2" {
		t.Errorf("expected writeConcern '2', got '%s'", cs.Options.WriteConcern)
	}
}

// TestParse_TLSCertificateKeyFileAlias tests alternative TLS cert file option name
func TestParse_TLSCertificateKeyFileAlias(t *testing.T) {
	uri := "laura://localhost?tlsCertificateKeyFile=/path/to/cert.pem"
	cs, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cs.Options.TLSCertFile != "/path/to/cert.pem" {
		t.Errorf("expected tlsCertFile '/path/to/cert.pem', got '%s'", cs.Options.TLSCertFile)
	}
}

// Benchmark tests
func BenchmarkParse_Simple(b *testing.B) {
	uri := "laura://localhost:8080/mydb"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(uri)
	}
}

func BenchmarkParse_Complex(b *testing.B) {
	uri := "mongodb://user:pass@host1:27017,host2:27017,host3:27017/mydb?replicaSet=rs0&maxConnections=100&timeout=5000&tls=true"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(uri)
	}
}

func BenchmarkString(b *testing.B) {
	cs, _ := Parse("laura://user:pass@host1:8080,host2:8081/mydb")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cs.String()
	}
}
