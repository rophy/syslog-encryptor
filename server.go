package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

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
	
	parser := NewMessageParser(conn, '\x00')
	
	for {
		message, err := parser.ReadMessage()
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from Unix connection: %v", err)
			}
			break
		}

		if err := s.processUnixMessage(message); err != nil {
			log.Printf("Error processing Unix message: %v", err)
		}
	}
}

func (s *UnixSyslogServer) processUnixMessage(data []byte) error {
	// Record metrics for processed message
	RecordProcessedLog(len(data))
	
	// Message already has correct format (\n preserved, \x00 discarded by parser)
	if err := encryptAndOutput(s.encryptor, data); err != nil {
		return fmt.Errorf("failed to encrypt and output message: %w", err)
	}
	
	return nil
}

