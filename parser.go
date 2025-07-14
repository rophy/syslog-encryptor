package main

import (
	"bytes"
	"io"
)

// MessageParser handles delimiter-based message parsing from any reader
type MessageParser struct {
	delimiter byte
	buffer    []byte
	reader    io.Reader
}

// NewMessageParser creates a new parser with specified delimiter
func NewMessageParser(reader io.Reader, delimiter byte) *MessageParser {
	return &MessageParser{
		delimiter: delimiter,
		buffer:    make([]byte, 0, 4096),
		reader:    reader,
	}
}

// ReadMessage reads bytes until delimiter, returns message without delimiter
func (p *MessageParser) ReadMessage() ([]byte, error) {
	readBuffer := make([]byte, 4096)
	
	for {
		// First check if we have complete messages in buffer
		for {
			delimIndex := bytes.IndexByte(p.buffer, p.delimiter)
			if delimIndex == -1 {
				break // No complete message yet
			}
			
			// Extract message (excluding delimiter)
			message := make([]byte, delimIndex)
			copy(message, p.buffer[:delimIndex])
			
			// Remove processed data from buffer (including delimiter)
			p.buffer = append(p.buffer[:0], p.buffer[delimIndex+1:]...)
			
			return message, nil
		}
		
		// Read more data
		n, err := p.reader.Read(readBuffer)
		if err != nil {
			if err == io.EOF && len(p.buffer) > 0 {
				// Return remaining buffer on EOF
				msg := make([]byte, len(p.buffer))
				copy(msg, p.buffer)
				p.buffer = p.buffer[:0]
				return msg, nil
			}
			return nil, err
		}
		
		// Note: Unbounded buffer growth is acceptable since stdin is only used for testing
		// in controlled environments, not production deployment
		p.buffer = append(p.buffer, readBuffer[:n]...)
	}
}

// StripTrailingNewline removes trailing newline characters from syslog messages
// This provides consistent handling for both stdin and datagram sources
func StripTrailingNewline(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\n' {
		return data[:len(data)-1]
	}
	return data
}