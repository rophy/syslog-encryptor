package main

import (
	"flag"
	"log"
	"log/syslog"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Command line flags
	var numLogs = flag.Int("n", 1000, "Number of log messages to generate")
	var logSize = flag.Int("s", 100, "Size of each log message in bytes")
	var tag = flag.String("tag", "syslog-generator", "Syslog tag/program name")
	var delay = flag.Duration("d", 0, "Delay between messages (e.g., 10ms, 1s)")
	flag.Parse()

	log.Printf("Starting syslog generator...")
	log.Printf("Messages: %d", *numLogs)
	log.Printf("Message size: %d bytes", *logSize)
	log.Printf("Tag: %s", *tag)
	log.Printf("Delay: %v", *delay)

	// Open syslog connection
	writer, err := syslog.New(syslog.LOG_INFO|syslog.LOG_USER, *tag)
	if err != nil {
		log.Fatalf("Failed to open syslog: %v", err)
	}
	defer writer.Close()

	// Generate a base message template
	baseMsg := "Log message "
	if *logSize <= len(baseMsg) {
		log.Fatalf("Log size must be greater than %d bytes", len(baseMsg))
	}

	// Calculate padding needed
	paddingSize := *logSize - len(baseMsg) - 10 // Reserve space for counter

	// Generate padding string (using repeated 'A' characters)
	padding := strings.Repeat("A", paddingSize)

	log.Printf("Generating %d syslog messages...", *numLogs)
	start := time.Now()

	// Generate log messages
	for i := 0; i < *numLogs; i++ {
		// Create message with exact size
		counter := strconv.Itoa(i + 1)
		
		// Adjust padding to maintain exact message size
		actualPadding := padding
		if len(baseMsg)+len(counter)+len(actualPadding) > *logSize {
			actualPadding = actualPadding[:*logSize-len(baseMsg)-len(counter)]
		} else if len(baseMsg)+len(counter)+len(actualPadding) < *logSize {
			needed := *logSize - len(baseMsg) - len(counter) - len(actualPadding)
			actualPadding += strings.Repeat("B", needed)
		}
		
		message := baseMsg + counter + actualPadding
		
		// Send to syslog
		if err := writer.Info(message); err != nil {
			log.Printf("Failed to send message %d: %v", i+1, err)
			continue
		}

		// Progress indicator
		if (i+1)%1000 == 0 {
			log.Printf("Sent %d messages...", i+1)
		}

		// Optional delay
		if *delay > 0 {
			time.Sleep(*delay)
		}
	}

	elapsed := time.Since(start)
	rate := float64(*numLogs) / elapsed.Seconds()

	log.Printf("Completed!")
	log.Printf("Sent %d messages in %v", *numLogs, elapsed)
	log.Printf("Rate: %.2f messages/second", rate)
	log.Printf("Throughput: %.2f KB/s", rate*float64(*logSize)/1024)
}