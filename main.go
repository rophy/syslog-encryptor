package main

import (
	"encoding/hex"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Set log output to stderr to keep stdout clean for JSON
	log.SetOutput(os.Stderr)
	
	// Configuration from environment variables (only if explicitly configured)
	listenAddr := os.Getenv("LISTEN_ADDR")

	// Support Unix socket for direct syslog integration (only if explicitly configured)
	socketPath := os.Getenv("SOCKET_PATH")

	encryptorPrivateKeyHex := os.Getenv("ENCRYPTOR_PRIVATE_KEY")
	if encryptorPrivateKeyHex == "" {
		log.Fatal("ENCRYPTOR_PRIVATE_KEY environment variable is required (32-byte hex string)")
	}

	decryptorPublicKeyHex := os.Getenv("DECRYPTOR_PUBLIC_KEY")
	if decryptorPublicKeyHex == "" {
		log.Fatal("DECRYPTOR_PUBLIC_KEY environment variable is required (32-byte hex string)")
	}

	// Decode encryptor private key
	encryptorPrivateKeyBytes, err := hex.DecodeString(encryptorPrivateKeyHex)
	if err != nil {
		log.Fatalf("Invalid ENCRYPTOR_PRIVATE_KEY format: %v", err)
	}
	if len(encryptorPrivateKeyBytes) != 32 {
		log.Fatalf("ENCRYPTOR_PRIVATE_KEY must be exactly 32 bytes (64 hex characters)")
	}

	var encryptorPrivateKey [32]byte
	copy(encryptorPrivateKey[:], encryptorPrivateKeyBytes)

	// Decode decryptor public key
	decryptorPublicKeyBytes, err := hex.DecodeString(decryptorPublicKeyHex)
	if err != nil {
		log.Fatalf("Invalid DECRYPTOR_PUBLIC_KEY format: %v", err)
	}
	if len(decryptorPublicKeyBytes) != 32 {
		log.Fatalf("DECRYPTOR_PUBLIC_KEY must be exactly 32 bytes (64 hex characters)")
	}

	var decryptorPublicKey [32]byte
	copy(decryptorPublicKey[:], decryptorPublicKeyBytes)

	// Create encryptor with configured private key
	encryptor, err := NewEncryptor(encryptorPrivateKey)
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}

	// Setup shared secret with decryptor public key
	if err := encryptor.SetupSharedSecret(decryptorPublicKey); err != nil {
		log.Fatalf("Failed to setup shared secret: %v", err)
	}

	// Log our public key for the decryptor to use
	log.Printf("Encryptor public key: %x", encryptor.GetPublicKey())
	log.Printf("Decryptor public key: %x", decryptorPublicKey)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		os.Exit(0)
	}()

	log.Printf("Starting MariaDB audit log encryptor...")

	// Validate that at least one server type is configured
	if listenAddr == "" && socketPath == "" {
		log.Fatal("At least one of LISTEN_ADDR or SOCKET_PATH must be defined")
	}

	// Start TCP server only if LISTEN_ADDR is explicitly defined
	if listenAddr != "" {
		go func() {
			log.Printf("Starting TCP syslog server on %s", listenAddr)
			server := NewSyslogServer(listenAddr, encryptor)
			if err := server.Start(); err != nil {
				log.Printf("TCP server failed: %v", err)
			}
		}()
	} else {
		log.Printf("LISTEN_ADDR not defined - TCP server disabled")
	}

	// Start Unix socket server only if SOCKET_PATH is explicitly defined
	if socketPath != "" {
		log.Printf("Starting Unix socket syslog server on %s", socketPath)
		unixServer := NewUnixSyslogServer(socketPath, encryptor)
		if err := unixServer.Start(); err != nil {
			log.Fatalf("Unix socket server failed: %v", err)
		}
	} else {
		log.Printf("SOCKET_PATH not defined - Unix socket server disabled")
		// Keep TCP server running indefinitely
		select {}
	}
}