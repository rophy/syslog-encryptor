package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type EncryptedLogEntry struct {
	Timestamp     string `json:"t"`
	Nonce         string `json:"n"`
	EncryptedData string `json:"m"`
	PublicKey     string `json:"k"`
}

func main() {
	// Configuration from environment variables
	decryptorPrivateKeyHex := os.Getenv("DECRYPTOR_PRIVATE_KEY")
	if decryptorPrivateKeyHex == "" {
		log.Fatal("DECRYPTOR_PRIVATE_KEY environment variable is required (32-byte hex string)")
	}

	encryptorPublicKeyHex := os.Getenv("ENCRYPTOR_PUBLIC_KEY")
	if encryptorPublicKeyHex == "" {
		log.Fatal("ENCRYPTOR_PUBLIC_KEY environment variable is required (32-byte hex string)")
	}

	// Decode decryptor private key
	decryptorPrivateKeyBytes, err := hex.DecodeString(decryptorPrivateKeyHex)
	if err != nil {
		log.Fatalf("Invalid DECRYPTOR_PRIVATE_KEY format: %v", err)
	}
	if len(decryptorPrivateKeyBytes) != 32 {
		log.Fatalf("DECRYPTOR_PRIVATE_KEY must be exactly 32 bytes (64 hex characters)")
	}

	var decryptorPrivateKey [32]byte
	copy(decryptorPrivateKey[:], decryptorPrivateKeyBytes)

	// Decode encryptor public key
	encryptorPublicKeyBytes, err := hex.DecodeString(encryptorPublicKeyHex)
	if err != nil {
		log.Fatalf("Invalid ENCRYPTOR_PUBLIC_KEY format: %v", err)
	}
	if len(encryptorPublicKeyBytes) != 32 {
		log.Fatalf("ENCRYPTOR_PUBLIC_KEY must be exactly 32 bytes (64 hex characters)")
	}

	var encryptorPublicKey [32]byte
	copy(encryptorPublicKey[:], encryptorPublicKeyBytes)

	// Create decryptor with configured private key
	decryptor, err := NewDecryptor(decryptorPrivateKey)
	if err != nil {
		log.Fatalf("Failed to create decryptor: %v", err)
	}

	// Setup shared secret with encryptor public key
	if err := decryptor.SetupSharedSecret(encryptorPublicKey); err != nil {
		log.Fatalf("Failed to setup shared secret: %v", err)
	}

	// Log key information to stderr (so it doesn't interfere with stdout)
	log.Printf("Decryptor public key: %x", decryptor.GetPublicKey())
	log.Printf("Encryptor public key: %x", encryptorPublicKey)
	log.Printf("Starting syslog decryptor - reading from stdin...")

	// Process stdin line by line
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		
		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse JSON
		var entry EncryptedLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			log.Printf("Error parsing JSON: %v", err)
			continue
		}

		// Decrypt the message
		decryptedMessage, err := decryptor.Decrypt(entry.Nonce, entry.EncryptedData)
		if err != nil {
			log.Printf("Error decrypting message: %v", err)
			continue
		}

		// Output the original log message to stdout
		fmt.Println(decryptedMessage)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading from stdin: %v", err)
	}
}