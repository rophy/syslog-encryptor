# Decryptor

A companion tool for syslog-encryptor that decrypts encrypted audit logs.

## Features

- Reads encrypted JSON lines from stdin
- Uses X25519 + AES-GCM decryption  
- Outputs original log messages to stdout
- Single static binary for easy deployment

## Configuration

Environment variables:

- `DECRYPTOR_PRIVATE_KEY`: 32-byte hex-encoded private key of the decryptor (required)
- `ENCRYPTOR_PUBLIC_KEY`: 32-byte hex-encoded public key of the encryptor (required)

## Usage

### Build

```bash
go build -o decryptor .
```

### Run

```bash
export DECRYPTOR_PRIVATE_KEY="your_decryptor_private_key_hex"
export ENCRYPTOR_PUBLIC_KEY="your_encryptor_public_key_hex"

# Decrypt from file
cat encrypted_logs.jsonl | ./decryptor

# Decrypt from encryptor output
docker logs syslog-encryptor | ./decryptor

# Real-time decryption
docker logs -f syslog-encryptor | ./decryptor
```

### Docker

```bash
docker build -t decryptor .
docker run -i -e DECRYPTOR_PRIVATE_KEY="your_private_key" \
              -e ENCRYPTOR_PUBLIC_KEY="your_public_key" \
              decryptor < encrypted_logs.jsonl
```

## Key Generation

Use the same key pairs generated for the encryptor:

```bash
# Use decryptor's private key and encryptor's public key
DECRYPTOR_PRIVATE_HEX=$(openssl pkey -in decryptor_private.pem -noout -text | grep -A3 "priv:" | tail -n+2 | tr -d ' :\n')
ENCRYPTOR_PUBLIC_HEX=$(openssl pkey -in encryptor_public.pem -pubin -noout -text | grep -A3 "pub:" | tail -n+2 | tr -d ' :\n')

export DECRYPTOR_PRIVATE_KEY=$DECRYPTOR_PRIVATE_HEX
export ENCRYPTOR_PUBLIC_KEY=$ENCRYPTOR_PUBLIC_HEX
```

## Input Format

Expects JSON lines with format:

```json
{
  "t": "2024-01-15T10:30:45.123456789Z",
  "n": "AQIDBAUGBwgJCgsMDQ4PEA==",
  "m": "ZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkwYWJjZGVmZ2hpams=",
  "k": "1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f809"
}
```

## Output

Original unencrypted log messages, one per line.