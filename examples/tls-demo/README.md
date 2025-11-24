# TLS/SSL Demo

This example demonstrates how to use TLS/SSL encryption with LauraDB's HTTP server.

## What This Demo Shows

1. **Generate Self-Signed Certificate**: How to create a self-signed TLS certificate for development
2. **Start HTTPS Server**: How to configure and start the server with TLS enabled
3. **Connect to HTTPS Server**: How to make secure HTTPS connections
4. **Verify Certificate Properties**: How to inspect certificate details
5. **Security Comparison**: Benefits of TLS vs plain HTTP

## Running the Demo

```bash
cd examples/tls-demo
go run main.go
```

## Key Features

### Self-Signed Certificates

The demo generates a self-signed certificate using the `server.GenerateSelfSignedCert()` function:

```go
err := server.GenerateSelfSignedCert(certFile, keyFile, "localhost")
```

This creates:
- An ECDSA P-256 private key
- A self-signed X.509 certificate valid for 1 year
- DNS entries for "localhost" and "127.0.0.1"

### TLS Configuration

Enable TLS by setting these configuration options:

```go
config := server.DefaultConfig()
config.EnableTLS = true
config.TLSCertFile = "/path/to/cert.pem"
config.TLSKeyFile = "/path/to/key.pem"
```

### HTTPS Client

Connect to the HTTPS server using a properly configured client:

```go
client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            // For self-signed certs (development only!)
            InsecureSkipVerify: true,
        },
    },
}
```

## Production Considerations

### Certificate Management

1. **Use CA-Signed Certificates**: In production, use certificates from a trusted Certificate Authority (Let's Encrypt, DigiCert, etc.)
2. **Never Skip Verification**: Remove `InsecureSkipVerify: true` in production clients
3. **Secure Private Keys**: Keep private keys secure, never commit to version control
4. **Regular Renewal**: Set up automatic certificate renewal before expiration

### Command-Line Usage

Start the server with TLS from the command line:

```bash
# Generate certificate (one-time setup)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Start server with TLS
./bin/laura-server -tls -tls-cert cert.pem -tls-key key.pem -port 8443
```

### Let's Encrypt Integration

For production use with Let's Encrypt:

```bash
# Obtain certificate using certbot
certbot certonly --standalone -d yourdomain.com

# Start server with Let's Encrypt certificate
./bin/laura-server \
  -tls \
  -tls-cert /etc/letsencrypt/live/yourdomain.com/fullchain.pem \
  -tls-key /etc/letsencrypt/live/yourdomain.com/privkey.pem \
  -port 443
```

## Security Benefits

### With TLS Enabled:
- ✓ All data encrypted in transit (queries, documents, credentials)
- ✓ Server authentication prevents impersonation
- ✓ Protection against man-in-the-middle attacks
- ✓ Data integrity verification

### Without TLS:
- ✗ Data transmitted in plain text
- ✗ Vulnerable to packet sniffing
- ✗ No server verification
- ✗ Susceptible to tampering

## TLS Configuration Options

All available TLS options:

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `EnableTLS` | bool | false | Enable/disable TLS |
| `TLSCertFile` | string | "" | Path to TLS certificate file (.pem) |
| `TLSKeyFile` | string | "" | Path to TLS private key file (.pem) |

## Example Output

```
LauraDB TLS/SSL Demo
====================

Demo 1: Generate Self-Signed Certificate
-----------------------------------------
Generating certificate for localhost...
✓ Certificate generated: /tmp/laura-tls-demo-123/cert.pem
✓ Private key generated: /tmp/laura-tls-demo-123/key.pem

Demo 2: Start HTTPS Server
--------------------------
🔒 TLS/SSL enabled
📜 Certificate: /tmp/laura-tls-demo-123/cert.pem
🚀 LauraDB server starting on https://localhost:8443
📁 Data directory: /tmp/laura-tls-demo-123/data
💾 Buffer pool size: 1000 pages

Demo 3: Connect to HTTPS Server
-------------------------------
Making HTTPS request to: https://localhost:8443/_health
✓ HTTPS connection successful!
  Status: 200
  TLS Version: TLS 1.3
  Cipher Suite: TLS_AES_128_GCM_SHA256
  Server Response: {"ok":true,"result":{"status":"healthy","time":"2025-11-24T..."}}

Demo 4: Verify Certificate Properties
-------------------------------------
Certificate Information:
  Subject: localhost
  Issuer: localhost
  Valid From: 2025-11-24T08:00:00Z
  Valid Until: 2026-11-24T08:00:00Z
  DNS Names: [localhost 127.0.0.1]
  Serial Number: 123456789...
```

## See Also

- [HTTP API Documentation](../../docs/http-api.md)
- [Performance Tuning Guide](../../docs/performance-tuning.md)
- [Authentication Demo](../auth-demo/)
