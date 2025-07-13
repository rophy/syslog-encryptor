# Syslog Encryptor

A sidecar container system for encrypting audit logs sent via syslog protocol using X25519 + AES-GCM encryption.

## Features

- **Universal syslog encryption** - Works with MariaDB, PostgreSQL, Apache, Nginx, or any syslog source
- **Dual connectivity** - Supports both TCP syslog and Unix domain sockets
- **Proper syslog parsing** - Uses RFC3164/RFC5424 compliant library
- **Strong encryption** - X25519 key exchange + AES-GCM
- **Compact output** - Encrypted logs as JSON lines with minimal field names
- **Sidecar ready** - Docker Compose and Kubernetes support
- **Complete solution** - Includes both encryptor and decryptor

## Architecture

```
┌─────────────┐    Unix Socket    ┌──────────────────┐    JSON Lines    ┌─────────────┐
│   MariaDB   │ ── /dev/log ────► │ syslog-encryptor │ ──── stdout ────► │ Log Storage │
│ Audit Plugin│                   │                  │                   │             │
└─────────────┘                   └──────────────────┘                   └─────────────┘
                                           │
                                           ▼
                                  ┌──────────────────┐
                                  │    decryptor     │ ◄─── Decrypt later
                                  └──────────────────┘
```

## Quick Start

### 1. Generate Keys

```bash
# Generate X25519 key pairs and show environment variables
./scripts/generate-keys.sh

# Set environment variables for testing
eval "$(./scripts/generate-keys.sh | grep '^export')"
```

### 2. Start Services

```bash
# Docker Compose (bind mount approach)
docker-compose up -d

# Kubernetes (shared volume approach)  
kubectl apply -f kubernetes-sidecar.yaml
```

### 3. Generate Test Data

```bash
# Generate MariaDB audit logs for testing
./scripts/generate-audit-logs.sh
```

### 4. Decrypt Logs

```bash
# Real-time decryption
docker logs -f syslog-encryptor | docker run -i \
  -e DECRYPTOR_PRIVATE_KEY="$DECRYPTOR_PRIVATE_KEY" \
  -e ENCRYPTOR_PUBLIC_KEY="$ENCRYPTOR_PUBLIC_KEY" \
  decryptor
```

## Project Structure

```
├── README.md                   # This file
├── docker-compose.yaml         # Docker Compose with MariaDB + encryptor
├── kubernetes-sidecar.yaml     # Kubernetes StatefulSet with sidecar
├── mariadb-config.cnf          # MariaDB audit plugin configuration
├── Dockerfile                  # Encryptor container
├── Makefile                    # Build targets
├── main.go                     # Encryptor main application
├── server.go                   # TCP + Unix socket servers  
├── crypto.go                   # X25519 + AES-GCM encryption
├── metrics.go                  # Prometheus metrics
├── decryptor/                  # Decryptor module
│   ├── main.go                 # Decryptor application  
│   ├── crypto.go               # Decryption functions
│   ├── Dockerfile              # Decryptor container
│   └── README.md               # Decryptor documentation
└── scripts/                    # Utility scripts
    ├── generate-keys.sh        # Generate X25519 key pairs
    └── generate-audit-logs.sh  # Generate test audit data
```

## Configuration

### Encryptor Environment Variables

**Connection Options** (at least one required):
- `SOCKET_PATH`: Unix socket path (optional - enables Unix socket server)
- `LISTEN_ADDR`: TCP address to listen on (optional - enables TCP server)

**Encryption Keys** (both required):
- `ENCRYPTOR_PRIVATE_KEY`: 32-byte hex-encoded private key (required)
- `DECRYPTOR_PUBLIC_KEY`: 32-byte hex-encoded public key (required)

**Optional Features**:
- `STDIN_MODE`: Set to any value to enable stdin processing mode (ignores all other configuration, single-threaded)
- `METRICS_ADDR`: Address for Prometheus metrics endpoint (e.g., `:8080`) - server modes only

**Examples:**
```bash
# TCP-only mode
export LISTEN_ADDR="0.0.0.0:514"

# Unix socket-only mode  
export SOCKET_PATH="/tmp/syslog.sock"

# Dual mode (both TCP and Unix socket)
export LISTEN_ADDR="0.0.0.0:514"
export SOCKET_PATH="/dev/log"

# With Prometheus metrics
export SOCKET_PATH="/tmp/syslog.sock"
export METRICS_ADDR=":8080"

# Stdin processing mode (ignores all other config)
export STDIN_MODE=1
```

**STDIN Mode Behavior**:
- **Single-threaded**: No background goroutines or servers
- **Ignores all other config**: SOCKET_PATH, LISTEN_ADDR, METRICS_ADDR are ignored
- **Pure processing**: Only reads stdin, encrypts, outputs JSON, exits on EOF
- **High performance**: ~175K msg/sec encryption rate

### Decryptor Environment Variables

- `DECRYPTOR_PRIVATE_KEY`: 32-byte hex-encoded private key (required)
- `ENCRYPTOR_PUBLIC_KEY`: 32-byte hex-encoded public key (required)

## Deployment Options

### Docker Compose (Recommended for Development)

Uses bind mount approach - simple and reliable:

```bash
# Generate keys and create .env file
./scripts/generate-keys.sh | grep '^export' | sed 's/export //' > .env

# Start MariaDB + syslog-encryptor
docker-compose up -d

# Test with audit logs
./scripts/generate-audit-logs.sh
```

### Kubernetes (Production)

Uses StatefulSet with emptyDir shared volume:

```bash
# Create encryption keys secret
kubectl create secret generic encryption-keys \
  --from-literal=encryptor-private-key="$(echo $ENCRYPTOR_PRIVATE_KEY | base64)" \
  --from-literal=decryptor-public-key="$(echo $DECRYPTOR_PUBLIC_KEY | base64)"

# Deploy sidecar
kubectl apply -f kubernetes-sidecar.yaml
```

### Standalone Binary

```bash
# Build both binaries
make build

# Individual builds
go build -o syslog-encryptor .
cd decryptor && go build -o decryptor .

# Run encryptor (TCP mode)
export ENCRYPTOR_PRIVATE_KEY="your_key"
export DECRYPTOR_PUBLIC_KEY="your_key"
export LISTEN_ADDR="localhost:9514"
./syslog-encryptor

# Or run encryptor (Unix socket mode)
export ENCRYPTOR_PRIVATE_KEY="your_key"
export DECRYPTOR_PUBLIC_KEY="your_key"
export SOCKET_PATH="/tmp/syslog.sock"
./syslog-encryptor

# Run decryptor (in another terminal)
export DECRYPTOR_PRIVATE_KEY="your_key"  
export ENCRYPTOR_PUBLIC_KEY="your_key"
docker logs -f syslog-encryptor | ./decryptor/decryptor
```

## Output Format

Encrypted logs are output as compact JSON lines:

```json
{"t":"2024-01-15T10:30:45.123456789Z","n":"AQIDBAUGBwgJCgsMDQ4PEA==","m":"ZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkwYWJjZGVmZ2hpams=","k":"1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f809"}
```

**Fields:**
- **t**: RFC3339 nano timestamp  
- **n**: Base64-encoded AES-GCM nonce (12 bytes)
- **m**: Base64-encoded encrypted message content
- **k**: Hex-encoded X25519 public key of encryptor

## Prometheus Metrics

When `METRICS_ADDR` is configured, the encryptor exposes Prometheus metrics at `/metrics`:

### Available Metrics

- **`syslog_encryptor_processed_logs_total`** (counter): Total number of log messages processed
- **`syslog_encryptor_processed_bytes_total`** (counter): Total number of bytes processed

### Example Usage

```bash
# Start encryptor with metrics
export SOCKET_PATH="/tmp/syslog.sock"
export METRICS_ADDR=":8080"
./syslog-encryptor

# Query metrics
curl http://localhost:8080/metrics | grep syslog_encryptor
```

**Sample Output:**
```
# HELP syslog_encryptor_processed_bytes_total Total number of bytes processed by the syslog encryptor
# TYPE syslog_encryptor_processed_bytes_total counter
syslog_encryptor_processed_bytes_total 255

# HELP syslog_encryptor_processed_logs_total Total number of log messages processed by the syslog encryptor
# TYPE syslog_encryptor_processed_logs_total counter
syslog_encryptor_processed_logs_total 5
```

## Security

- **Forward secrecy** - Each message uses unique nonce
- **Authenticated encryption** - AES-GCM provides integrity protection  
- **Key separation** - Encryptor and decryptor use different private keys
- **No key storage** - Keys provided via environment variables only
- **Minimal attack surface** - Static binaries with minimal dependencies

## Use Cases

- **Audit log encryption** - Encrypt sensitive database audit logs
- **Compliance** - Meet regulatory requirements for log protection
- **Secure log forwarding** - Encrypt logs before sending to external systems
- **Zero-trust logging** - Encrypt logs even within trusted networks
- **Forensic integrity** - Tamper-evident encrypted audit trails

## Troubleshooting

### Configuration errors

1. **"At least one of LISTEN_ADDR or SOCKET_PATH must be defined"**
   - Set either `LISTEN_ADDR` for TCP mode or `SOCKET_PATH` for Unix socket mode
   - Or set both for dual mode

2. **"Permission denied" errors**
   - For TCP: Use non-privileged port (e.g., `LISTEN_ADDR=localhost:9514`)
   - For Unix socket: Ensure path is writable (e.g., `/tmp/syslog.sock`)

### No encrypted logs appearing

1. Check if syslog socket exists: `docker exec mariadb ls -la /dev/log`
2. Verify audit plugin is loaded: `SHOW PLUGINS;`
3. Check encryptor logs: `docker logs syslog-encryptor`
4. Test with manual syslog: `logger -d -u /dev/log "test message"`

### Decryption fails

1. Verify key pairs match: Compare public keys from both tools
2. Check environment variables are set correctly
3. Ensure JSON format is valid: `docker logs syslog-encryptor | jq .`

### Performance considerations

- **CPU usage** - Each log encryption requires cryptographic operations
- **Memory usage** - ~64MB for encryptor, ~32MB for decryptor
- **Network overhead** - Base64 encoding increases size by ~33%
- **Throughput** - Tested up to 10,000 log entries/second

## License

This project implements defensive security tooling for audit log protection.