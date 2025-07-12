#!/bin/bash

# Syslog Encryptor Key Generation Helper
# Generates X25519 key pairs and outputs environment variables

set -e

ENCRYPTOR_PRIVATE_PEM="encryptor_private.pem"
DECRYPTOR_PRIVATE_PEM="decryptor_private.pem"

echo "🔐 Syslog Encryptor Key Generation Helper"
echo

# Generate encryptor private key if it doesn't exist
if [ ! -f "$ENCRYPTOR_PRIVATE_PEM" ]; then
    echo "📝 Generating encryptor private key: $ENCRYPTOR_PRIVATE_PEM"
    openssl genpkey -algorithm X25519 -out "$ENCRYPTOR_PRIVATE_PEM"
else
    echo "✅ Using existing encryptor private key: $ENCRYPTOR_PRIVATE_PEM"
fi

# Generate decryptor private key if it doesn't exist
if [ ! -f "$DECRYPTOR_PRIVATE_PEM" ]; then
    echo "📝 Generating decryptor private key: $DECRYPTOR_PRIVATE_PEM"
    openssl genpkey -algorithm X25519 -out "$DECRYPTOR_PRIVATE_PEM"
else
    echo "✅ Using existing decryptor private key: $DECRYPTOR_PRIVATE_PEM"
fi

echo

# Extract public keys (temporary files)
ENCRYPTOR_PUBLIC_PEM=$(mktemp)
DECRYPTOR_PUBLIC_PEM=$(mktemp)

openssl pkey -in "$ENCRYPTOR_PRIVATE_PEM" -pubout -out "$ENCRYPTOR_PUBLIC_PEM"
openssl pkey -in "$DECRYPTOR_PRIVATE_PEM" -pubout -out "$DECRYPTOR_PUBLIC_PEM"

# Extract hex values
echo "🔑 Extracting key material..."

ENCRYPTOR_PRIVATE_HEX=$(openssl pkey -in "$ENCRYPTOR_PRIVATE_PEM" -noout -text | grep -A3 "priv:" | tail -n+2 | tr -d ' :\n')
ENCRYPTOR_PUBLIC_HEX=$(openssl pkey -in "$ENCRYPTOR_PUBLIC_PEM" -pubin -noout -text | grep -A3 "pub:" | tail -n+2 | tr -d ' :\n')
DECRYPTOR_PRIVATE_HEX=$(openssl pkey -in "$DECRYPTOR_PRIVATE_PEM" -noout -text | grep -A3 "priv:" | tail -n+2 | tr -d ' :\n')
DECRYPTOR_PUBLIC_HEX=$(openssl pkey -in "$DECRYPTOR_PUBLIC_PEM" -pubin -noout -text | grep -A3 "pub:" | tail -n+2 | tr -d ' :\n')

# Clean up temporary files
rm "$ENCRYPTOR_PUBLIC_PEM" "$DECRYPTOR_PUBLIC_PEM"

echo "✅ Key generation complete!"
echo
echo "📋 Environment Variables:"
echo "=========================="
echo
echo "# For encryptor"
echo "export ENCRYPTOR_PRIVATE_KEY=\"$ENCRYPTOR_PRIVATE_HEX\""
echo "export DECRYPTOR_PUBLIC_KEY=\"$DECRYPTOR_PUBLIC_HEX\""
echo
echo "# For decryptor"
echo "export DECRYPTOR_PRIVATE_KEY=\"$DECRYPTOR_PRIVATE_HEX\""
echo "export ENCRYPTOR_PUBLIC_KEY=\"$ENCRYPTOR_PUBLIC_HEX\""
echo
