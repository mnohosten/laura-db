# TLS/SSL Encryption

LauraDB supports TLS/SSL encryption for secure communication between clients and the HTTP server. This guide covers certificate management, server configuration, and security best practices.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Certificate Management](#certificate-management)
- [Server Configuration](#server-configuration)
- [Client Configuration](#client-configuration)
- [Production Deployment](#production-deployment)
- [Security Best Practices](#security-best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

### What is TLS/SSL?

Transport Layer Security (TLS) and its predecessor Secure Sockets Layer (SSL) are cryptographic protocols that provide secure communication over a network. TLS/SSL ensures:

- **Encryption**: All data transmitted between client and server is encrypted
- **Authentication**: Server identity is verified using digital certificates
- **Integrity**: Data cannot be tampered with during transmission

### When to Use TLS/SSL

Enable TLS/SSL when:

- ✓ Transmitting sensitive data (credentials, personal information, financial data)
- ✓ Deploying on public networks or the internet
- ✓ Compliance requirements mandate encryption (HIPAA, PCI DSS, GDPR)
- ✓ Protecting against man-in-the-middle attacks

You might skip TLS for:

- ✗ Local development on trusted networks
- ✗ Internal networks with other security measures
- ✗ Performance-critical scenarios where encryption overhead is prohibitive

## Quick Start

### 1. Generate Self-Signed Certificate (Development)

For development and testing, generate a self-signed certificate:

```go
import "github.com/mnohosten/laura-db/pkg/server"

// Generate certificate for localhost
err := server.GenerateSelfSignedCert("cert.pem", "key.pem", "localhost")
if err != nil {
    log.Fatal(err)
}
```

Or use OpenSSL:

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

### 2. Start Server with TLS

```go
config := server.DefaultConfig()
config.EnableTLS = true
config.TLSCertFile = "cert.pem"
config.TLSKeyFile = "key.pem"
config.Port = 8443  // Standard HTTPS development port

srv, err := server.New(config)
if err != nil {
    log.Fatal(err)
}

srv.Start()
```

Or via command line:

```bash
./bin/laura-server -tls -tls-cert cert.pem -tls-key key.pem -port 8443
```

### 3. Connect from Client

```go
import (
    "crypto/tls"
    "net/http"
)

client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            // Only for self-signed certs in development!
            InsecureSkipVerify: true,
        },
    },
}

resp, err := client.Get("https://localhost:8443/_health")
```

## Certificate Management

### Types of Certificates

#### 1. Self-Signed Certificates

**Use for**: Development, testing, internal tools

**Pros**:
- Free
- Quick to generate
- Full control

**Cons**:
- Not trusted by browsers/clients by default
- No third-party validation
- Not suitable for production

**Generate using LauraDB**:

```go
import "github.com/mnohosten/laura-db/pkg/server"

err := server.GenerateSelfSignedCert(
    "cert.pem",      // Certificate file path
    "key.pem",       // Private key file path
    "localhost",     // Common Name (hostname)
)
```

The generated certificate:
- Uses ECDSA P-256 elliptic curve cryptography
- Valid for 1 year (365 days)
- Includes DNS entries for the specified host
- Automatically adds localhost/127.0.0.1 for local development

#### 2. CA-Signed Certificates

**Use for**: Production, public-facing services

**Pros**:
- Trusted by all browsers/clients
- Third-party validation
- Professional appearance

**Cons**:
- Cost (unless using Let's Encrypt)
- Renewal process
- Domain validation required

**Obtain from**:
- Let's Encrypt (free, automated)
- DigiCert, GlobalSign, Comodo (paid)
- Internal enterprise CA

### Let's Encrypt Integration

Let's Encrypt provides free, automated TLS certificates.

#### Installation

```bash
# Install certbot
# macOS
brew install certbot

# Ubuntu/Debian
apt-get install certbot

# CentOS/RHEL
yum install certbot
```

#### Obtain Certificate

```bash
# Standalone mode (stops other web servers)
certbot certonly --standalone -d yourdomain.com -d www.yourdomain.com

# Certificates stored in:
# /etc/letsencrypt/live/yourdomain.com/fullchain.pem
# /etc/letsencrypt/live/yourdomain.com/privkey.pem
```

#### Configure LauraDB

```bash
./bin/laura-server \
  -tls \
  -tls-cert /etc/letsencrypt/live/yourdomain.com/fullchain.pem \
  -tls-key /etc/letsencrypt/live/yourdomain.com/privkey.pem \
  -port 443 \
  -host 0.0.0.0
```

#### Auto-Renewal

Let's Encrypt certificates expire every 90 days. Set up auto-renewal:

```bash
# Add to crontab
0 0 * * * certbot renew --post-hook "systemctl restart laura-db"

# Or use systemd timer
systemctl enable certbot-renew.timer
```

### Certificate Formats

LauraDB expects PEM-encoded certificates and keys.

#### Convert from other formats:

```bash
# DER to PEM (certificate)
openssl x509 -inform der -in cert.der -out cert.pem

# PKCS12 to PEM (certificate + key)
openssl pkcs12 -in cert.p12 -out cert.pem -clcerts -nokeys
openssl pkcs12 -in cert.p12 -out key.pem -nocerts -nodes

# PFX to PEM
openssl pkcs12 -in cert.pfx -out cert.pem -nokeys
openssl pkcs12 -in cert.pfx -out key.pem -nocerts -nodes
```

## Server Configuration

### Configuration Options

```go
type Config struct {
    // ... other fields ...

    EnableTLS   bool   // Enable/disable TLS
    TLSCertFile string // Path to certificate file (.pem)
    TLSKeyFile  string // Path to private key file (.pem)
}
```

### Example Configurations

#### Development (Self-Signed)

```go
config := server.DefaultConfig()
config.Host = "localhost"
config.Port = 8443
config.EnableTLS = true
config.TLSCertFile = "./dev-cert.pem"
config.TLSKeyFile = "./dev-key.pem"
```

#### Production (Let's Encrypt)

```go
config := server.DefaultConfig()
config.Host = "0.0.0.0"  // Listen on all interfaces
config.Port = 443         // Standard HTTPS port
config.EnableTLS = true
config.TLSCertFile = "/etc/letsencrypt/live/yourdomain.com/fullchain.pem"
config.TLSKeyFile = "/etc/letsencrypt/live/yourdomain.com/privkey.pem"
```

#### Production (Commercial CA)

```go
config := server.DefaultConfig()
config.Host = "0.0.0.0"
config.Port = 443
config.EnableTLS = true
config.TLSCertFile = "/etc/ssl/certs/yourdomain.com.pem"
config.TLSKeyFile = "/etc/ssl/private/yourdomain.com-key.pem"
```

### Command-Line Flags

```bash
# Enable TLS
./bin/laura-server -tls -tls-cert cert.pem -tls-key key.pem

# Specify port (default: 8080, recommended for HTTPS: 443 or 8443)
./bin/laura-server -tls -tls-cert cert.pem -tls-key key.pem -port 8443

# Full example
./bin/laura-server \
  -host 0.0.0.0 \
  -port 443 \
  -tls \
  -tls-cert /etc/ssl/certs/cert.pem \
  -tls-key /etc/ssl/private/key.pem \
  -data-dir /var/lib/laura-db \
  -buffer-size 10000
```

## Client Configuration

### Go Client

#### Production (Verify Certificates)

```go
import (
    "crypto/tls"
    "net/http"
)

client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            // Use system's root CA certificates
            // This is the default and recommended for production
        },
    },
}

resp, err := client.Get("https://yourdomain.com/_health")
```

#### Development (Self-Signed Certificates)

```go
client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            // WARNING: Only use in development!
            InsecureSkipVerify: true,
        },
    },
}

resp, err := client.Get("https://localhost:8443/_health")
```

#### Custom CA Certificate

```go
import (
    "crypto/tls"
    "crypto/x509"
    "io/ioutil"
)

// Load CA certificate
caCert, err := ioutil.ReadFile("ca-cert.pem")
if err != nil {
    log.Fatal(err)
}

caCertPool := x509.NewCertPool()
caCertPool.AppendCertsFromPEM(caCert)

client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            RootCAs: caCertPool,
        },
    },
}
```

### cURL

```bash
# Production (verify certificate)
curl https://yourdomain.com/_health

# Development (skip verification - insecure!)
curl -k https://localhost:8443/_health

# Specify CA certificate
curl --cacert ca-cert.pem https://localhost:8443/_health
```

### Browser

Browsers automatically verify certificates using the system's root CA store. For self-signed certificates:

1. Navigate to `https://localhost:8443`
2. Click "Advanced" or similar
3. Click "Proceed to localhost (unsafe)" or similar

**Note**: This should only be done in development. Never bypass certificate warnings in production.

## Production Deployment

### Checklist

- [ ] Use certificates from a trusted CA (Let's Encrypt or commercial)
- [ ] Configure automatic certificate renewal
- [ ] Use strong TLS configuration (TLS 1.2+, strong cipher suites)
- [ ] Secure private key file (chmod 600, restricted access)
- [ ] Never commit private keys to version control
- [ ] Enable HSTS (HTTP Strict Transport Security) headers
- [ ] Monitor certificate expiration
- [ ] Configure firewall to allow HTTPS (port 443)
- [ ] Set up certificate revocation if needed

### Systemd Service

```ini
# /etc/systemd/system/laura-db.service
[Unit]
Description=LauraDB Server
After=network.target

[Service]
Type=simple
User=laura-db
Group=laura-db
WorkingDirectory=/opt/laura-db
ExecStart=/opt/laura-db/bin/laura-server \
  -host 0.0.0.0 \
  -port 443 \
  -tls \
  -tls-cert /etc/letsencrypt/live/yourdomain.com/fullchain.pem \
  -tls-key /etc/letsencrypt/live/yourdomain.com/privkey.pem \
  -data-dir /var/lib/laura-db
Restart=always
RestartSec=10

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/laura-db

[Install]
WantedBy=multi-user.target
```

### Nginx Reverse Proxy

For advanced setups, use Nginx as a reverse proxy:

```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;

    # Strong SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # HSTS
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    location / {
        proxy_pass http://127.0.0.1:8080;  # LauraDB without TLS
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}
```

## Security Best Practices

### Certificate Security

1. **Protect Private Keys**
   ```bash
   # Set restrictive permissions
   chmod 600 /path/to/key.pem
   chown laura-db:laura-db /path/to/key.pem

   # Never commit to version control
   echo "*.pem" >> .gitignore
   ```

2. **Regular Rotation**
   - Rotate certificates before expiration
   - Let's Encrypt: auto-renew every 60 days (expires at 90)
   - Commercial CAs: typically 1-2 year validity

3. **Secure Storage**
   - Use encrypted filesystems for key storage
   - Consider Hardware Security Modules (HSMs) for critical deployments
   - Use secret management tools (HashiCorp Vault, AWS Secrets Manager)

### TLS Configuration

1. **Modern TLS Only**
   ```go
   // Go's http.Server uses secure defaults
   // TLS 1.2 and 1.3 are enabled by default
   // Weak ciphers are excluded
   ```

2. **Cipher Suites**

   Go automatically selects secure cipher suites. The defaults are recommended.

3. **HSTS Headers**

   Add via middleware or reverse proxy:
   ```
   Strict-Transport-Security: max-age=31536000; includeSubDomains
   ```

### Monitoring

1. **Certificate Expiration**
   ```bash
   # Check expiration date
   openssl x509 -in cert.pem -noout -enddate

   # Days until expiration
   echo | openssl s_client -connect localhost:8443 2>/dev/null | \
     openssl x509 -noout -checkend 2592000  # 30 days
   ```

2. **SSL Labs Testing**

   Test your deployment: https://www.ssllabs.com/ssltest/

3. **Monitoring Tools**
   - Prometheus with certificate expiration exporter
   - Nagios SSL certificate check plugin
   - Custom scripts with alerting

## Troubleshooting

### Common Issues

#### 1. "Certificate file not found"

**Error**: `TLS certificate file not found: cert.pem`

**Solution**:
- Verify file path is correct
- Check file permissions (readable by server process)
- Use absolute paths in production

#### 2. "Certificate is expired"

**Error**: `x509: certificate has expired`

**Solution**:
- Renew certificate: `certbot renew`
- Generate new self-signed cert for development
- Check system clock (time sync issues)

#### 3. "Connection refused" on HTTPS

**Error**: Client cannot connect

**Solution**:
- Verify server is running: `curl https://localhost:8443/_health`
- Check firewall allows port 443: `sudo ufw allow 443/tcp`
- Verify TLS is enabled in config
- Check server logs for startup errors

#### 4. "Certificate is not trusted"

**Warning**: Browser shows security warning

**Solution**:
- **Production**: Use CA-signed certificate
- **Development**: Add exception in browser
- **Corporate**: Install internal CA certificate

#### 5. "Private key does not match certificate"

**Error**: `tls: private key does not match public key`

**Solution**:
- Ensure cert and key are from the same pair
- Regenerate both together
- Check for file corruption

### Debug Commands

```bash
# Test TLS connection
openssl s_client -connect localhost:8443 -showcerts

# Verify certificate
openssl x509 -in cert.pem -text -noout

# Check private key
openssl rsa -in key.pem -check

# Verify cert-key pair match
openssl x509 -noout -modulus -in cert.pem | openssl md5
openssl rsa -noout -modulus -in key.pem | openssl md5
# (These should output the same hash)

# Test specific TLS version
openssl s_client -connect localhost:8443 -tls1_2
openssl s_client -connect localhost:8443 -tls1_3
```

## Performance Considerations

### TLS Overhead

TLS adds computational overhead:
- Initial handshake: ~1-2ms
- Encryption/decryption: ~5-10% CPU overhead
- Memory: Minimal (~10-50KB per connection)

### Optimization

1. **Use TLS 1.3**
   - Faster handshake (1-RTT vs 2-RTT)
   - Better cipher suites
   - Enabled by default in Go

2. **Connection Pooling**
   - Reuse TLS connections
   - HTTP Keep-Alive enabled by default

3. **Hardware Acceleration**
   - Use CPU with AES-NI support
   - Consider SSL offload hardware for high-volume deployments

## References

- [Go crypto/tls Documentation](https://pkg.go.dev/crypto/tls)
- [Let's Encrypt](https://letsencrypt.org/)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
- [TLS 1.3 Specification](https://www.rfc-editor.org/rfc/rfc8446)

## See Also

- [HTTP API Documentation](http-api.md)
- [Authentication Guide](authentication.md)
- [Performance Tuning](performance-tuning.md)
- [TLS Demo Example](../examples/tls-demo/)
