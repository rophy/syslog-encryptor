package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

type EncryptedLogEntry struct {
	Timestamp     string `json:"t"`
	Nonce         string `json:"n"`
	EncryptedData string `json:"m"`
	PublicKey     string `json:"k"`
}

// Unix Socket Server for direct syslog integration
type UnixSyslogServer struct {
	encryptor  *Encryptor
	socketPath string
}

func NewUnixSyslogServer(socketPath string, encryptor *Encryptor) *UnixSyslogServer {
	return &UnixSyslogServer{
		encryptor:  encryptor,
		socketPath: socketPath,
	}
}

func (s *UnixSyslogServer) Start() error {
	// Remove existing socket file if it exists
	if err := os.RemoveAll(s.socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create Unix domain socket
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create Unix socket: %w", err)
	}
	defer listener.Close()

	// Set socket permissions so MariaDB can write to it
	if err := os.Chmod(s.socketPath, 0666); err != nil {
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	log.Printf("Unix syslog encryptor listening on %s", s.socketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting Unix connection: %v", err)
			continue
		}

		go s.handleUnixConnection(conn)
	}
}

func (s *UnixSyslogServer) handleUnixConnection(conn net.Conn) {
	defer conn.Close()
	
	buffer := make([]byte, 4096)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("Error reading from Unix connection: %v", err)
			}
			break
		}

		if err := s.processUnixMessage(buffer[:n]); err != nil {
			log.Printf("Error processing Unix message: %v", err)
		}
	}
}

func (s *UnixSyslogServer) processUnixMessage(data []byte) error {
	message := string(data)
	
	// Encrypt the message content
	encryptResult, err := s.encryptor.Encrypt(message)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}

	// Record metrics for processed message
	RecordProcessedLog(len(message))

	// Create encrypted log entry
	entry := EncryptedLogEntry{
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Nonce:         encryptResult.Nonce,
		EncryptedData: encryptResult.EncryptedData,
		PublicKey:     fmt.Sprintf("%x", s.encryptor.GetPublicKey()),
	}

	// Output as JSON line to stdout
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}