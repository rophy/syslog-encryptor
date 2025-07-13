package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Set log output to stderr to keep stdout clean for JSON
	log.SetOutput(os.Stderr)
	
	// Support Unix socket for direct syslog integration (required unless STDIN_MODE)
	socketPath := os.Getenv("SOCKET_PATH")
	
	// Support stdin processing mode (only if explicitly configured)
	stdinMode := os.Getenv("STDIN_MODE") != ""
	
	// Prometheus metrics endpoint configuration
	metricsAddr := os.Getenv("METRICS_ADDR")

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

	// Handle stdin mode first - ignore all other configuration
	if stdinMode {
		log.Printf("Starting stdin processing mode...")
		if err := processStdinSimple(encryptor); err != nil {
			log.Fatalf("Stdin processing failed: %v", err)
		}
		return
	}

	log.Printf("Starting MariaDB audit log encryptor...")

	// Initialize Prometheus metrics (only for server modes)
	InitMetrics()
	
	// Start metrics server if configured
	if metricsAddr != "" {
		go func() {
			log.Printf("Starting Prometheus metrics server on %s", metricsAddr)
			if err := StartMetricsServer(metricsAddr); err != nil {
				log.Printf("Metrics server failed: %v", err)
			}
		}()
	}

	// Validate that socket path is configured for server mode
	if socketPath == "" {
		log.Fatal("SOCKET_PATH environment variable is required")
	}

	// Start Unix socket server
	log.Printf("Starting Unix socket syslog server on %s", socketPath)
	unixServer := NewUnixSyslogServer(socketPath, encryptor)
	if err := unixServer.Start(); err != nil {
		log.Fatalf("Unix socket server failed: %v", err)
	}
}

// processStdin reads log lines from stdin and encrypts them
func processStdin(encryptor *Encryptor) error {
	scanner := bufio.NewScanner(os.Stdin)
	lineCount := 0
	
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		
		// Encrypt the line
		encryptResult, err := encryptor.Encrypt(line)
		if err != nil {
			log.Printf("Failed to encrypt line %d: %v", lineCount, err)
			continue
		}
		
		// Record metrics for processed message
		RecordProcessedLog(len(line))
		
		// Create encrypted log entry (same format as server.go)
		entry := EncryptedLogEntry{
			Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
			Nonce:         encryptResult.Nonce,
			EncryptedData: encryptResult.EncryptedData,
			PublicKey:     fmt.Sprintf("%x", encryptor.GetPublicKey()),
		}
		
		// Output as JSON line to stdout
		jsonData, err := json.Marshal(entry)
		if err != nil {
			log.Printf("Failed to marshal JSON for line %d: %v", lineCount, err)
			continue
		}
		
		fmt.Println(string(jsonData))
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stdin: %w", err)
	}
	
	log.Printf("Processed %d lines from stdin", lineCount)
	return nil
}

// processStdinSimple reads log lines from stdin and encrypts them (simple single-threaded mode)
func processStdinSimple(encryptor *Encryptor) error {
	scanner := bufio.NewScanner(os.Stdin)
	lineCount := 0
	
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		
		// Encrypt the line
		encryptResult, err := encryptor.Encrypt(line)
		if err != nil {
			log.Printf("Failed to encrypt line %d: %v", lineCount, err)
			continue
		}
		
		// Create encrypted log entry (no metrics recording)
		entry := EncryptedLogEntry{
			Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
			Nonce:         encryptResult.Nonce,
			EncryptedData: encryptResult.EncryptedData,
			PublicKey:     fmt.Sprintf("%x", encryptor.GetPublicKey()),
		}
		
		// Output as JSON line to stdout
		jsonData, err := json.Marshal(entry)
		if err != nil {
			log.Printf("Failed to marshal JSON for line %d: %v", lineCount, err)
			continue
		}
		
		fmt.Println(string(jsonData))
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stdin: %w", err)
	}
	
	log.Printf("Processed %d lines from stdin", lineCount)
	return nil
}