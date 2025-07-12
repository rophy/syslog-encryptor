package main

import (
	"encoding/hex"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Configuration from environment variables
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "0.0.0.0:514"
	}

	// Support Unix socket for direct syslog integration
	socketPath := os.Getenv("SOCKET_PATH")
	if socketPath == "" {
		socketPath = "/dev/log"
	}

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

	// Start both TCP and Unix socket servers
	go func() {
		log.Printf("Starting TCP syslog server on %s", listenAddr)
		server := NewSyslogServer(listenAddr, encryptor)
		if err := server.Start(); err != nil {
			log.Printf("TCP server failed: %v", err)
		}
	}()

	// Start Unix socket server for direct syslog integration
	log.Printf("Starting Unix socket syslog server on %s", socketPath)
	unixServer := NewUnixSyslogServer(socketPath, encryptor)
	if err := unixServer.Start(); err != nil {
		log.Fatalf("Unix socket server failed: %v", err)
	}
}