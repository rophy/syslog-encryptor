package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

// Unix Socket Server for direct syslog integration
type UnixSyslogServer struct {
	encryptor  *Encryptor
	socketPath string
	listener   net.PacketConn
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

	// Create Unix domain datagram socket (SOCK_DGRAM)
	listener, err := net.ListenPacket("unixgram", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create Unix datagram socket: %w", err)
	}
	s.listener = listener
	defer s.Cleanup()

	// Set socket permissions so applications can write to it
	if err := os.Chmod(s.socketPath, 0666); err != nil {
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	log.Printf("Unix syslog encryptor listening on %s (SOCK_DGRAM)", s.socketPath)

	// Handle datagram packets
	buffer := make([]byte, 65536) // Max UDP packet size
	for {
		n, addr, err := listener.ReadFrom(buffer)
		if err != nil {
			log.Printf("Error reading from Unix datagram socket: %v", err)
			continue
		}

		// Process the packet in a goroutine
		go s.handleUnixPacket(buffer[:n], addr)
	}
}

func (s *UnixSyslogServer) handleUnixPacket(data []byte, addr net.Addr) {
	// For SOCK_DGRAM, each packet is a complete message
	// Use consistent newline handling
	data = StripTrailingNewline(data)

	if err := s.processUnixMessage(data); err != nil {
		log.Printf("Error processing Unix packet from %v: %v", addr, err)
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

// Cleanup closes the listener and removes the socket file
func (s *UnixSyslogServer) Cleanup() {
	if s.listener != nil {
		log.Printf("Closing Unix datagram socket...")
		s.listener.Close()
	}
	
	if s.socketPath != "" {
		log.Printf("Removing socket file: %s", s.socketPath)
		if err := os.RemoveAll(s.socketPath); err != nil {
			log.Printf("Error removing socket file: %v", err)
		}
	}
}
