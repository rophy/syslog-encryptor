# Syslog Encryptor Makefile

.PHONY: build build-docker clean test-logs help

# Default target
all: build

# Build both binaries
build:
	@echo "🔨 Building encryptor binary..."
	go build -o syslog-encryptor .
	@echo "🔨 Building decryptor binary..."
	cd decryptor && go build -o decryptor .
	@echo "✅ Both binaries built successfully"

# Build both Docker containers
build-docker:
	@echo "🐳 Building encryptor container..."
	docker build -t syslog-encryptor .
	@echo "🐳 Building decryptor container..."
	docker build -t decryptor decryptor/
	@echo "✅ Both containers built successfully"

# Clean built binaries
clean:
	@echo "🧹 Cleaning up binaries..."
	rm -f syslog-encryptor
	rm -f decryptor/decryptor
	@echo "✅ Cleanup complete"

# Test encryption/decryption flow
test-logs:
	@echo "🔄 Testing encryption/decryption flow..."
	@echo "📝 Make sure docker-compose is running and .pem files exist"
	docker logs syslog-encryptor | docker run -i --rm \
		-e DECRYPTOR_PRIVATE_KEY="$$(openssl pkey -in decryptor_private.pem -noout -text | grep -A3 'priv:' | tail -n+2 | tr -d ' :\n')" \
		-e ENCRYPTOR_PUBLIC_KEY="$$(openssl pkey -in encryptor_private.pem -pubout | openssl pkey -pubin -noout -text | grep -A3 'pub:' | tail -n+2 | tr -d ' :\n')" \
		decryptor

# Show help
help:
	@echo "Syslog Encryptor Build Targets:"
	@echo "  build        - Build both encryptor and decryptor binaries"
	@echo "  build-docker - Build both encryptor and decryptor containers"
	@echo "  clean        - Remove built binaries"
	@echo "  test-logs    - Pipe encryptor logs to decryptor for testing"
	@echo "  help         - Show this help message"