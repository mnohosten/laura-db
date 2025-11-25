# Encryption at Rest

LauraDB provides transparent encryption at rest to protect sensitive data stored on disk. All database files and write-ahead logs can be encrypted using industry-standard AES-256 encryption.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Encryption Algorithms](#encryption-algorithms)
- [Getting Started](#getting-started)
- [API Reference](#api-reference)
- [Performance Considerations](#performance-considerations)
- [Security Best Practices](#security-best-practices)
- [Limitations](#limitations)

## Overview

Encryption at rest protects data when it's stored on disk. LauraDB encrypts:
- **Data pages**: All document data stored in data.db files
- **Write-Ahead Log (WAL)**: Transaction logs in wal.log files

The encryption is transparent - encrypted data is automatically decrypted when read and encrypted when written.

## Features

- **AES-256 Encryption**: Industry-standard symmetric encryption
- **Multiple Algorithms**: Support for AES-GCM (with authentication) and AES-CTR
- **Password-Based Key Derivation**: PBKDF2 with 100,000 iterations
- **Authenticated Encryption**: GCM mode provides both confidentiality and authenticity
- **Transparent Operation**: No changes to database API - encryption handled automatically
- **Per-Database Keys**: Each database can use different encryption keys

## Encryption Algorithms

### AES-256-GCM (Recommended)

**Algorithm**: AES-256 in Galois/Counter Mode
**Authentication**: Yes (built-in)
**Security**: Provides both confidentiality and authenticity
**Overhead**: ~28 bytes per page (12-byte nonce + 16-byte auth tag)

```go
config, err := encryption.NewConfigFromPassword("my-password", encryption.AlgorithmAES256GCM)
```

**Advantages:**
- Detects tampering and corruption
- Prevents unauthorized modifications
- Industry standard for authenticated encryption

**Use Cases:**
- Production databases with sensitive data
- Compliance requirements (HIPAA, GDPR, PCI-DSS)
- High-security applications

### AES-256-CTR

**Algorithm**: AES-256 in Counter Mode
**Authentication**: No
**Security**: Provides confidentiality only
**Overhead**: ~16 bytes per page (16-byte IV)

```go
config, err := encryption.NewConfigFromPassword("my-password", encryption.AlgorithmAES256CTR)
```

**Advantages:**
- Slightly less overhead than GCM
- Faster on some platforms

**Use Cases:**
- When authentication is provided by other means
- Performance-critical applications with lower security requirements

### No Encryption

```go
config := encryption.DefaultConfig() // AlgorithmNone
```

Use when encryption is not required (development, testing, or public data).

## Getting Started

### Basic Usage with Password

```go
package main

import (
    "github.com/mnohosten/laura-db/pkg/encryption"
    "github.com/mnohosten/laura-db/pkg/storage"
)

func main() {
    // Create encryption config from password
    config, err := encryption.NewConfigFromPassword(
        "my-secure-password-123",
        encryption.AlgorithmAES256GCM,
    )
    if err != nil {
        panic(err)
    }

    // Create encrypted disk manager
    edm, err := encryption.NewEncryptedDiskManager(
        "/path/to/data.db",
        config,
    )
    if err != nil {
        panic(err)
    }
    defer edm.Close()

    // Use normally - encryption is automatic
    pageID, _ := edm.AllocatePage()
    page := storage.NewPage(pageID, storage.PageTypeData)

    // Write data (automatically encrypted)
    copy(page.Data, []byte("Secret data"))
    edm.WritePage(page)

    // Read data (automatically decrypted)
    readPage, _ := edm.ReadPage(pageID)
    // readPage.Data contains decrypted data
}
```

### Using Explicit Keys

For more control, provide an explicit 32-byte encryption key:

```go
// Generate a secure random key
key := make([]byte, 32)
_, err := rand.Read(key)
if err != nil {
    panic(err)
}

// Create config with explicit key
config, err := encryption.NewConfigFromKey(key, encryption.AlgorithmAES256GCM)
if err != nil {
    panic(err)
}

// Store the key securely (e.g., in a key management system)
// DO NOT hardcode keys in source code
```

### Encrypting Write-Ahead Log

```go
// Create encrypted WAL
config, _ := encryption.NewConfigFromPassword("wal-password", encryption.AlgorithmAES256GCM)
wal, err := encryption.NewEncryptedWAL("/path/to/wal.log", config)
if err != nil {
    panic(err)
}
defer wal.Close()

// Append log records (automatically encrypted)
record := &storage.LogRecord{
    Type:   storage.LogRecordInsert,
    TxnID:  1,
    PageID: 0,
    Data:   []byte("Transaction data"),
}
lsn, _ := wal.Append(record)

// Replay WAL (automatically decrypted)
records, _ := wal.Replay()
```

## API Reference

### Configuration

#### `NewConfigFromPassword(password string, algorithm Algorithm) (*Config, error)`

Creates encryption config with key derived from password using PBKDF2.

**Parameters:**
- `password`: Password for key derivation (must not be empty)
- `algorithm`: Encryption algorithm to use

**Returns:** Encryption config or error

**Example:**
```go
config, err := encryption.NewConfigFromPassword("secure-pass-123", encryption.AlgorithmAES256GCM)
```

#### `NewConfigFromKey(key []byte, algorithm Algorithm) (*Config, error)`

Creates encryption config with an explicit 32-byte key.

**Parameters:**
- `key`: 32-byte encryption key (for AES-256)
- `algorithm`: Encryption algorithm to use

**Returns:** Encryption config or error

**Example:**
```go
key := []byte{/* 32 bytes */}
config, err := encryption.NewConfigFromKey(key, encryption.AlgorithmAES256GCM)
```

#### `DefaultConfig() *Config`

Returns default config with no encryption (AlgorithmNone).

### Encrypted Disk Manager

#### `NewEncryptedDiskManager(path string, config *Config) (*EncryptedDiskManager, error)`

Creates an encrypted disk manager for data files.

**Methods:**
- `ReadPage(pageID PageID) (*Page, error)` - Read and decrypt page
- `WritePage(page *Page) error` - Encrypt and write page
- `AllocatePage() (PageID, error)` - Allocate new page
- `Sync() error` - Flush to disk
- `Close() error` - Close disk manager
- `Stats() map[string]interface{}` - Get statistics

### Encrypted WAL

#### `NewEncryptedWAL(path string, config *Config) (*EncryptedWAL, error)`

Creates an encrypted write-ahead log.

**Methods:**
- `Append(record *LogRecord) (uint64, error)` - Append encrypted log record
- `Replay() ([]*LogRecord, error)` - Replay and decrypt all records
- `Checkpoint() error` - Create checkpoint
- `Flush() error` - Flush to disk
- `Close() error` - Close WAL

## Performance Considerations

### Encryption Overhead

| Algorithm | Overhead per Page | Relative Speed |
|-----------|------------------|----------------|
| None | 0 bytes | 1.0x (baseline) |
| AES-256-CTR | ~16 bytes | ~0.95x |
| AES-256-GCM | ~28 bytes | ~0.90x |

**Benchmarks** (4KB pages):
```
BenchmarkEncryptGCM-8    50000    24532 ns/op    ~163 MB/s
BenchmarkDecryptGCM-8    50000    23891 ns/op    ~167 MB/s
BenchmarkEncryptCTR-8    60000    19845 ns/op    ~206 MB/s
BenchmarkDecryptCTR-8    60000    19234 ns/op    ~212 MB/s
```

### Performance Tips

1. **Use CTR for Better Performance**: If authentication isn't critical
2. **Buffer Pool Caching**: Encrypted data is cached in buffer pool (no re-decryption)
3. **Hardware Acceleration**: Modern CPUs have AES-NI instructions for faster encryption
4. **Batch Operations**: Group multiple writes to reduce encryption overhead
5. **Key Caching**: Encryption keys are cached (no re-derivation on each operation)

### Memory Usage

- **Keys**: 32 bytes per database
- **Nonces/IVs**: Generated per page (not stored in memory)
- **Cipher State**: ~200 bytes per active encryptor

## Security Best Practices

### Key Management

✅ **DO:**
- Use strong, unique passwords (16+ characters)
- Store keys in secure key management systems (AWS KMS, HashiCorp Vault, etc.)
- Rotate keys periodically (requires re-encryption)
- Use different keys for different databases
- Use environment variables or config files (never hardcode keys)

❌ **DON'T:**
- Hardcode passwords in source code
- Commit keys to version control
- Reuse keys across multiple databases
- Use weak or default passwords
- Store keys alongside encrypted data

### Password Guidelines

```go
// Good: Strong password
config, _ := encryption.NewConfigFromPassword(
    "7x$mK9#pL2@nQ5!rT8^vW3&yU6*zA4",  // Strong, random
    encryption.AlgorithmAES256GCM,
)

// Bad: Weak password
config, _ := encryption.NewConfigFromPassword(
    "password123",  // Too simple, easily guessable
    encryption.AlgorithmAES256GCM,
)
```

### Key Derivation

LauraDB uses PBKDF2 with SHA-256 for password-based key derivation:
- **Iterations**: 100,000 (recommended by NIST)
- **Salt**: 32 bytes random salt per password
- **Output**: 32-byte key for AES-256

### Algorithm Selection

**For Production:**
```go
// Recommended: AES-256-GCM (provides authentication)
config, _ := encryption.NewConfigFromPassword(password, encryption.AlgorithmAES256GCM)
```

**For High Performance:**
```go
// Alternative: AES-256-CTR (faster, but no authentication)
config, _ := encryption.NewConfigFromPassword(password, encryption.AlgorithmAES256CTR)
```

### Threat Model

Encryption at rest protects against:
- ✅ Disk theft or physical access
- ✅ Unauthorized file access
- ✅ Backup media compromise
- ✅ Cloud storage exposure

Encryption at rest does NOT protect against:
- ❌ Memory dumps (data is decrypted in memory)
- ❌ Running process attacks
- ❌ Application-level vulnerabilities
- ❌ Compromised encryption keys

## Limitations

### Current Limitations

1. **Page Size Overhead**: Encrypted pages have ~28-33 bytes overhead, reducing usable space
2. **No Migration Tool**: Existing databases must be manually migrated to use encryption
3. **Single Key**: All pages in a database use the same encryption key
4. **No Key Rotation**: Changing keys requires re-encryption of entire database
5. **Buffer Pool**: Data in buffer pool is unencrypted (in-memory plaintext)

### Future Enhancements

- [ ] Automatic migration tool for existing databases
- [ ] Key rotation support
- [ ] Per-collection encryption keys
- [ ] Encrypted buffer pool option
- [ ] Hardware security module (HSM) integration
- [ ] Key escrow and recovery mechanisms

## Examples

See the complete example program at `examples/encryption-demo/main.go` for demonstrations of:
- Basic encryption with password
- Algorithm comparison (GCM vs CTR)
- Encrypted WAL usage
- Wrong key protection

Run the example:
```bash
cd examples/encryption-demo
go run main.go
```

## Compliance

Encryption at rest helps meet compliance requirements for:
- **HIPAA** (Health Insurance Portability and Accountability Act)
- **GDPR** (General Data Protection Regulation)
- **PCI-DSS** (Payment Card Industry Data Security Standard)
- **SOC 2** (System and Organization Controls 2)
- **FIPS 140-2** (Federal Information Processing Standards)

**Note**: Compliance requires proper implementation of all security controls, not just encryption. Consult with security professionals for compliance audits.

## Troubleshooting

### Common Errors

**"cipher: message authentication failed"**
- Cause: Wrong encryption key or corrupted data
- Solution: Verify using correct password/key

**"encrypted data too large"**
- Cause: Page data exceeds available space after encryption overhead
- Solution: Reduce data size to account for ~33 bytes overhead

**"key must be 32 bytes"**
- Cause: Invalid key length for AES-256
- Solution: Use `NewConfigFromPassword` or provide 32-byte key

### Debugging

Enable encryption statistics:
```go
stats := edm.Stats()
fmt.Printf("Encryption: %v\n", stats["encryption_algorithm"])
fmt.Printf("Enabled: %v\n", stats["encryption_enabled"])
```

## References

- [NIST Special Publication 800-38D](https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38d.pdf) - Recommendation for Block Cipher Modes of Operation: Galois/Counter Mode (GCM)
- [PBKDF2 Specification (RFC 2898)](https://www.rfc-editor.org/rfc/rfc2898)
- [AES Encryption Standard (FIPS 197)](https://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.197.pdf)

## Support

For questions or issues with encryption at rest:
1. Check this documentation
2. Review example code in `examples/encryption-demo/`
3. Open an issue on GitHub
4. Consult security professionals for production deployments
