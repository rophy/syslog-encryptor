# Syslog Encryptor

A sidecar container for encrypting MariaDB audit logs sent via syslog protocol using X25519 + AES-GCM encryption.

## Features

- Listens for syslog messages on configurable port
- Uses proper syslog parsing library (RFC3164/RFC5424)
- Encrypts log content with X25519 key exchange + AES-GCM
- Outputs encrypted logs as JSON lines to stdout
- Single static binary for easy deployment

## Configuration

Environment variables:

- `LISTEN_ADDR`: Address to listen on (default: `0.0.0.0:514`)
- `ENCRYPTOR_PRIVATE_KEY`: 32-byte hex-encoded private key of the encryptor (required)
- `DECRYPTOR_PUBLIC_KEY`: 32-byte hex-encoded public key of the decryptor (required)

## Key Generation

Generate X25519 key pairs with OpenSSL:

```bash
# Generate encryptor keys
openssl genpkey -algorithm X25519 -out encryptor_private.pem
openssl pkey -in encryptor_private.pem -pubout -out encryptor_public.pem

# Generate decryptor keys  
openssl genpkey -algorithm X25519 -out decryptor_private.pem
openssl pkey -in decryptor_private.pem -pubout -out decryptor_public.pem

# Extract hex values for configuration
ENCRYPTOR_PRIVATE_HEX=$(openssl pkey -in encryptor_private.pem -noout -text | grep -A3 "priv:" | tail -n+2 | tr -d ' :\n')
DECRYPTOR_PUBLIC_HEX=$(openssl pkey -in decryptor_public.pem -pubin -noout -text | grep -A3 "pub:" | tail -n+2 | tr -d ' :\n')

echo "ENCRYPTOR_PRIVATE_KEY=$ENCRYPTOR_PRIVATE_HEX"
echo "DECRYPTOR_PUBLIC_KEY=$DECRYPTOR_PUBLIC_HEX"
```

## Usage

### Build

```bash
go build -o syslog-encryptor .
```

### Run

```bash
export ENCRYPTOR_PRIVATE_KEY="your_encryptor_private_key_hex"
export DECRYPTOR_PUBLIC_KEY="your_decryptor_public_key_hex"
export LISTEN_ADDR="0.0.0.0:514"
./syslog-encryptor
```

### Docker

```bash
docker build -t syslog-encryptor .
docker run -e ENCRYPTOR_PRIVATE_KEY="your_private_key" \
           -e DECRYPTOR_PUBLIC_KEY="your_public_key" \
           -p 514:514 syslog-encryptor
```

## Kubernetes Sidecar

Example sidecar configuration for MariaDB:

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: mariadb
    image: mariadb:latest
    # Configure MariaDB to send audit logs to localhost:514
  - name: syslog-encryptor
    image: syslog-encryptor:latest
    env:
    - name: ENCRYPTOR_PRIVATE_KEY
      valueFrom:
        secretKeyRef:
          name: encryption-keys
          key: encryptor-private-key
    - name: DECRYPTOR_PUBLIC_KEY
      valueFrom:
        secretKeyRef:
          name: encryption-keys
          key: decryptor-public-key
    - name: LISTEN_ADDR
      value: "0.0.0.0:514"
    ports:
    - containerPort: 514
```

## Output Format

Each encrypted log is output as a JSON line:

```json
{
  "t": "2024-01-15T10:30:45.123456789Z",
  "n": "AQIDBAUGBwgJCgsMDQ4PEA==",
  "m": "ZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkwYWJjZGVmZ2hpams=",
  "k": "1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f809"
}
```

Fields:
- **t**: RFC3339 nano precision timestamp when encryption occurred
- **n**: Base64-encoded AES-GCM nonce (12 bytes)
- **m**: Base64-encoded AES-GCM encrypted message content
- **k**: Hex-encoded X25519 public key of the encryptor