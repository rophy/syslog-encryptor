package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/influxdata/go-syslog/v3"
	"github.com/influxdata/go-syslog/v3/rfc3164"
	"github.com/influxdata/go-syslog/v3/rfc5424"
)

type EncryptedLogEntry struct {
	Timestamp     string `json:"t"`
	Nonce         string `json:"n"`
	EncryptedData string `json:"m"`
	PublicKey     string `json:"k"`
}

type SyslogServer struct {
	encryptor *Encryptor
	listen    string
}

func NewSyslogServer(listen string, encryptor *Encryptor) *SyslogServer {
	return &SyslogServer{
		encryptor: encryptor,
		listen:    listen,
	}
}

func (s *SyslogServer) parseMessage(data []byte) (syslog.Message, error) {
	// Try RFC5424 first, then fall back to RFC3164
	parser5424 := rfc5424.NewParser()
	if msg, err := parser5424.Parse(data); err == nil {
		return msg, nil
	}

	parser3164 := rfc3164.NewParser()
	return parser3164.Parse(data)
}

func (s *SyslogServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	buffer := make([]byte, 4096)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("Error reading from connection: %v", err)
			}
			break
		}

		if err := s.processMessage(buffer[:n]); err != nil {
			log.Printf("Error processing message: %v", err)
		}
	}
}

func (s *SyslogServer) processMessage(data []byte) error {
	msg, err := s.parseMessage(data)
	if err != nil {
		return fmt.Errorf("failed to parse syslog message: %w", err)
	}

	// Extract message content
	var messageContent string
	
	// Type assert to access the Message field from the Base struct
	if rfc5424Msg, ok := msg.(*rfc5424.SyslogMessage); ok {
		if rfc5424Msg.Message != nil {
			messageContent = *rfc5424Msg.Message
		}
	} else if rfc3164Msg, ok := msg.(*rfc3164.SyslogMessage); ok {
		if rfc3164Msg.Message != nil {
			messageContent = *rfc3164Msg.Message
		}
	}
	
	// Fallback to raw data if message extraction failed
	if messageContent == "" {
		messageContent = string(data)
	}

	// Encrypt the message content
	encryptResult, err := s.encryptor.Encrypt(messageContent)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}

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

func (s *SyslogServer) Start() error {
	listener, err := net.Listen("tcp", s.listen)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	defer listener.Close()

	log.Printf("Syslog encryptor listening on %s", s.listen)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
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