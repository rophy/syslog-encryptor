# Syslog Encryptor Makefile

.PHONY: build build-docker clean help

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

# Show help
help:
	@echo "Syslog Encryptor Build Targets:"
	@echo "  build        - Build both encryptor and decryptor binaries"
	@echo "  build-docker - Build both encryptor and decryptor containers"
	@echo "  clean        - Remove built binaries"
	@echo "  help         - Show this help message"