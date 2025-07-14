#!/bin/bash

# Syslog Encryptor Socket Performance Test
# Uses Docker to test socket performance with syslog-generator

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Configuration
NUM_LINES=1000000
LINE_LENGTH=1024
OUTPUT_FILE="/tmp/speedtest_socket_output.txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Syslog Encryptor Socket Speed Test${NC}"
echo "===================================="
echo "Messages to send: $NUM_LINES"
echo "Message length: $LINE_LENGTH bytes"
echo "Total test data: $(( NUM_LINES * LINE_LENGTH / 1024 / 1024 )) MB"
echo

# Cleanup function
cleanup() {
    echo -e "${YELLOW}🧹 Cleaning up Docker containers...${NC}"
    cd "$PROJECT_DIR/tests/socket"
    docker compose down --volumes 2>/dev/null || true
    cd "$PROJECT_DIR"
}
trap cleanup EXIT

# Check if binaries exist and build
if [[ ! -f "$PROJECT_DIR/syslog-encryptor" ]]; then
    echo -e "${RED}❌ syslog-encryptor binary not found. Run 'make build' first.${NC}"
    exit 1
fi

# Generate keys if they don't exist
echo -e "${BLUE}🔑 Setting up encryption keys...${NC}"
cd "$PROJECT_DIR"
eval "$(./scripts/generate-keys.sh | grep '^export')"

# Navigate to socket test directory
cd "$PROJECT_DIR/tests/socket"

# Update docker-compose for performance test
echo -e "${BLUE}🔧 Configuring performance test...${NC}"

# Create temporary docker-compose override for performance testing
cat > docker-compose.override.yml <<EOF
services:
  syslog-generator:
    command: ["sh", "-c", "while [ ! -S /syslog/test.sock ]; do echo 'Waiting for socket...' && sleep 0.1; done && echo 'Socket found, linking and starting performance test...' && ln -s /syslog/test.sock /dev/log && ./syslog-generator -n $NUM_LINES -s $LINE_LENGTH -tag speedtest -d 0ms"]
    depends_on:
      - syslog-encryptor
EOF

echo -e "${BLUE}🐳 Starting Docker containers...${NC}"

# Clean up any existing containers
docker compose down --volumes 2>/dev/null || true

# Record start time (will be more accurate once we see the generator start)
echo -e "${BLUE}⚡ Running socket speed test...${NC}"

# Start containers and capture output
docker compose up --build > "$OUTPUT_FILE.log" 2>&1 &
COMPOSE_PID=$!

# Wait for the test to complete by monitoring the log
echo "Monitoring test progress..."
timeout=120  # 2 minutes timeout
start_found=false
end_found=false

while [[ $timeout -gt 0 ]]; do
    if [[ -f "$OUTPUT_FILE.log" ]]; then
        # Look for generator completion
        if grep -q "Completed!" "$OUTPUT_FILE.log" 2>/dev/null; then
            end_found=true
            break
        fi
        
        # Look for generator start
        if ! $start_found && grep -q "Generating.*syslog messages" "$OUTPUT_FILE.log" 2>/dev/null; then
            start_found=true
            echo "✅ Generator started, test in progress..."
        fi
    fi
    
    sleep 1
    timeout=$((timeout - 1))
done

# Stop compose
kill $COMPOSE_PID 2>/dev/null || true
wait $COMPOSE_PID 2>/dev/null || true

if ! $end_found; then
    echo -e "${RED}❌ Test did not complete within timeout${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Test completed successfully${NC}"

# Get encrypted JSON output from the encryptor container logs
docker logs socket-test-encryptor 2>/dev/null | grep '{"t":' > "$OUTPUT_FILE" || touch "$OUTPUT_FILE"
output_lines=$(wc -l < "$OUTPUT_FILE" 2>/dev/null || echo 0)

# Calculate processing metrics from timestamps
echo
echo -e "${GREEN}📊 SOCKET SPEED TEST RESULTS${NC}"
echo "============================"

if [[ $output_lines -gt 0 ]]; then
    # Extract first and last timestamps
    first_timestamp=$(head -n1 "$OUTPUT_FILE" | jq -r '.t' 2>/dev/null)
    last_timestamp=$(tail -n1 "$OUTPUT_FILE" | jq -r '.t' 2>/dev/null)
    
    if [[ "$first_timestamp" != "null" && "$last_timestamp" != "null" && "$first_timestamp" != "" && "$last_timestamp" != "" ]]; then
        # Convert timestamps to epoch seconds with nanosecond precision
        first_epoch=$(date -d "$first_timestamp" +%s.%N 2>/dev/null)
        last_epoch=$(date -d "$last_timestamp" +%s.%N 2>/dev/null)
        
        if [[ -n "$first_epoch" && -n "$last_epoch" ]]; then
            # Calculate processing duration
            processing_time=$(echo "$last_epoch - $first_epoch" | bc -l 2>/dev/null)
            
            if [[ -n "$processing_time" ]] && (( $(echo "$processing_time > 0" | bc -l 2>/dev/null) )); then
                # Calculate metrics
                messages_per_second=$(echo "scale=2; $output_lines / $processing_time" | bc -l 2>/dev/null)
                total_bytes=$(( NUM_LINES * LINE_LENGTH ))
                mb_per_second=$(echo "scale=2; $total_bytes / $processing_time / 1024 / 1024" | bc -l 2>/dev/null)
                
                echo "Output lines: $output_lines"
                echo "Processing time: ${processing_time}s"
                echo "Messages per second: $messages_per_second"
                echo "Throughput: ${mb_per_second} MB/s"
                echo "Success rate: $(echo "scale=1; $output_lines * 100 / $NUM_LINES" | bc -l 2>/dev/null)%"
            else
                echo -e "${RED}❌ Could not calculate processing duration${NC}"
                echo "Output lines: $output_lines"
            fi
        else
            echo -e "${RED}❌ Could not parse timestamps${NC}"
            echo "Output lines: $output_lines"
        fi
    else
        echo -e "${RED}❌ Could not extract valid timestamps${NC}"
        echo "Output lines: $output_lines"
    fi
else
    echo -e "${RED}❌ No encrypted output captured${NC}"
fi

# Output verification
echo
echo -e "${BLUE}🔍 OUTPUT VERIFICATION${NC}"
echo "===================="

if [[ $output_lines -gt 0 ]]; then
    # Check if output contains valid JSON (without "k" field)
    valid_json=0
    sample_lines=100
    sample_size=$( [[ $output_lines -lt $sample_lines ]] && echo $output_lines || echo $sample_lines )
    
    while IFS= read -r line; do
        if echo "$line" | jq -e 'has("t") and has("n") and has("m")' > /dev/null 2>&1; then
            valid_json=$((valid_json + 1))
        fi
    done < <(head -n $sample_size "$OUTPUT_FILE")
    
    if [[ $valid_json -eq $sample_size ]]; then
        echo -e "${GREEN}✅ All sampled output lines are valid encrypted JSON${NC}"
    else
        echo -e "${YELLOW}⚠️  Some output lines may not be valid JSON (${valid_json}/${sample_size})${NC}"
    fi
    
    # Show sample output
    echo
    echo "Sample encrypted output:"
    head -n 3 "$OUTPUT_FILE"
else
    echo -e "${RED}❌ No encrypted output captured${NC}"
fi

# Performance analysis
echo
echo -e "${BLUE}📈 PERFORMANCE ANALYSIS${NC}"
echo "======================"

if [[ -n "$messages_per_second" ]] && [[ "$messages_per_second" != "" ]]; then
    # Performance tiers for socket processing
    if (( $(echo "$messages_per_second > 100000" | bc -l 2>/dev/null || echo 0) )); then
        echo -e "${GREEN}🚀 Excellent performance (>100K msgs/sec)${NC}"
    elif (( $(echo "$messages_per_second > 50000" | bc -l 2>/dev/null || echo 0) )); then
        echo -e "${GREEN}✅ Good performance (>50K msgs/sec)${NC}"
    elif (( $(echo "$messages_per_second > 10000" | bc -l 2>/dev/null || echo 0) )); then
        echo -e "${YELLOW}⚠️  Moderate performance (>10K msgs/sec)${NC}"
    else
        echo -e "${RED}🐌 Low performance (<10K msgs/sec)${NC}"
    fi
    
    if [[ -n "$processing_time" ]] && [[ "$output_lines" -gt 0 ]]; then
        avg_time=$(echo "scale=6; $processing_time / $output_lines * 1000" | bc -l 2>/dev/null || echo "N/A")
        echo "Average time per message: ${avg_time}ms"
    fi
else
    echo -e "${RED}❌ No performance data available${NC}"
fi

echo
echo -e "${BLUE}📁 Files preserved:${NC}"
echo "- Test log: $OUTPUT_FILE.log ($(du -h "$OUTPUT_FILE.log" 2>/dev/null | cut -f1 || echo "N/A"))"
echo "- Encrypted output: $OUTPUT_FILE ($(du -h "$OUTPUT_FILE" 2>/dev/null | cut -f1 || echo "N/A"))"
echo "- Files are kept for further analysis"

# Clean up docker-compose override
rm -f docker-compose.override.yml

echo
echo -e "${GREEN}✅ Socket performance test completed${NC}"
