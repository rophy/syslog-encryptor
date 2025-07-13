#!/bin/bash

# Syslog Encryptor Performance Stress Test
# Tests single core encryption performance without MariaDB dependency

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Configuration
SOCKET_PATH="/tmp/syslog-stress.sock"
TEST_DURATION=10  # seconds
MESSAGE_SIZE=200  # bytes
CONCURRENT_WRITERS=1  # Start with 1 for single core test

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Syslog Encryptor Stress Test${NC}"
echo "=================================="
echo "Socket Path: $SOCKET_PATH"
echo "Test Duration: ${TEST_DURATION}s"
echo "Message Size: ${MESSAGE_SIZE} bytes"
echo "Concurrent Writers: $CONCURRENT_WRITERS"
echo

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}🧹 Cleaning up...${NC}"
    if [[ -n "$ENCRYPTOR_PID" ]] && kill -0 "$ENCRYPTOR_PID" 2>/dev/null; then
        kill "$ENCRYPTOR_PID"
        wait "$ENCRYPTOR_PID" 2>/dev/null || true
    fi
    
    # Kill any remaining background jobs
    jobs -p | xargs -r kill 2>/dev/null || true
    
    rm -f "$SOCKET_PATH"
    rm -f /tmp/stress_*.log
    echo -e "${GREEN}✅ Cleanup complete${NC}"
}

trap cleanup EXIT INT TERM

# Check if binary exists
if [[ ! -f "$PROJECT_DIR/syslog-encryptor" ]]; then
    echo -e "${RED}❌ syslog-encryptor binary not found. Run 'make build' first.${NC}"
    exit 1
fi

# Generate keys if they don't exist
echo -e "${BLUE}🔑 Setting up encryption keys...${NC}"
cd "$PROJECT_DIR"
eval "$(./scripts/generate-keys.sh | grep '^export')"

# Start syslog-encryptor
echo -e "${BLUE}🔧 Starting syslog-encryptor...${NC}"
export SOCKET_PATH="$SOCKET_PATH"
./syslog-encryptor > /tmp/stress_output.log 2> /tmp/stress_errors.log &
ENCRYPTOR_PID=$!

# Wait for socket to be created
echo -e "${YELLOW}⏳ Waiting for socket creation...${NC}"
for i in {1..10}; do
    if [[ -S "$SOCKET_PATH" ]]; then
        break
    fi
    sleep 0.5
done

if [[ ! -S "$SOCKET_PATH" ]]; then
    echo -e "${RED}❌ Socket not created after 5 seconds${NC}"
    cat /tmp/stress_errors.log
    exit 1
fi

echo -e "${GREEN}✅ Socket created successfully${NC}"

# Generate test message of specified size
generate_message() {
    local size=$1
    local msg="<14>$(date '+%b %d %H:%M:%S') stress-test: "
    local padding_needed=$((size - ${#msg}))
    
    if [[ $padding_needed -gt 0 ]]; then
        printf "%s%*s" "$msg" $padding_needed "" | tr ' ' 'A'
    else
        printf "%.${size}s" "$msg"
    fi
}

# Function to send messages continuously
stress_sender() {
    local worker_id=$1
    local message_count=0
    local start_time=$(date +%s.%N)
    
    while [[ $(($(date +%s) - $(date -d @${start_time%.*} +%s))) -lt $TEST_DURATION ]]; do
        local message=$(generate_message $MESSAGE_SIZE)
        echo "$message" | socat - UNIX-CONNECT:"$SOCKET_PATH" 2>/dev/null || {
            echo "Worker $worker_id: Connection failed" >&2
            break
        }
        ((message_count++))
        
        # Brief pause to prevent overwhelming the socket
        sleep 0.001  # 1ms between messages
    done
    
    echo "$worker_id:$message_count" > "/tmp/stress_worker_${worker_id}.count"
}

# Start stress test
echo -e "${BLUE}⚡ Starting stress test for ${TEST_DURATION} seconds...${NC}"
echo -e "${YELLOW}📊 Monitoring CPU usage: top -p $ENCRYPTOR_PID${NC}"

# Start CPU monitoring in background
top -b -n $((TEST_DURATION + 5)) -d 1 -p $ENCRYPTOR_PID > /tmp/stress_cpu.log 2>/dev/null &
CPU_MONITOR_PID=$!

# Record start metrics
start_time=$(date +%s)
start_output_lines=$(wc -l < /tmp/stress_output.log 2>/dev/null || echo "0")

# Start concurrent message senders
echo -e "${BLUE}🔥 Starting $CONCURRENT_WRITERS message sender(s)...${NC}"
for i in $(seq 1 $CONCURRENT_WRITERS); do
    stress_sender $i &
done

# Wait for test duration
sleep $TEST_DURATION

# Kill all senders
jobs -p | grep -v $ENCRYPTOR_PID | grep -v $CPU_MONITOR_PID | xargs -r kill 2>/dev/null || true
wait

# Collect results
end_time=$(date +%s)
end_output_lines=$(wc -l < /tmp/stress_output.log 2>/dev/null || echo "0")
total_encrypted_lines=$((end_output_lines - start_output_lines))
actual_duration=$((end_time - start_time))

# Calculate total messages sent
total_sent=0
for i in $(seq 1 $CONCURRENT_WRITERS); do
    if [[ -f "/tmp/stress_worker_${i}.count" ]]; then
        worker_count=$(cut -d: -f2 "/tmp/stress_worker_${i}.count")
        total_sent=$((total_sent + worker_count))
    fi
done

# Display results
echo
echo -e "${GREEN}📊 STRESS TEST RESULTS${NC}"
echo "======================="
echo "Test Duration: ${actual_duration}s"
echo "Messages Sent: $total_sent"
echo "Messages Encrypted: $total_encrypted_lines"
echo "Send Rate: $(bc -l <<< "scale=2; $total_sent / $actual_duration") msg/sec"
echo "Encryption Rate: $(bc -l <<< "scale=2; $total_encrypted_lines / $actual_duration") msg/sec"
echo "Success Rate: $(bc -l <<< "scale=1; $total_encrypted_lines * 100 / $total_sent")%"
echo "Message Size: ${MESSAGE_SIZE} bytes"
echo "Total Data Encrypted: $(bc -l <<< "scale=2; $total_encrypted_lines * $MESSAGE_SIZE / 1024") KB"

# CPU usage analysis
if [[ -f /tmp/stress_cpu.log ]]; then
    echo
    echo -e "${BLUE}💻 CPU USAGE ANALYSIS${NC}"
    echo "==================="
    
    # Extract CPU percentages (skip header lines)
    cpu_values=$(grep -E "^\s*$ENCRYPTOR_PID" /tmp/stress_cpu.log | awk '{print $9}' | grep -E '^[0-9]+\.?[0-9]*$' || true)
    
    if [[ -n "$cpu_values" ]]; then
        avg_cpu=$(echo "$cpu_values" | awk '{sum+=$1; count++} END {if(count>0) print sum/count; else print 0}')
        max_cpu=$(echo "$cpu_values" | sort -n | tail -1)
        echo "Average CPU Usage: ${avg_cpu}%"
        echo "Peak CPU Usage: ${max_cpu}%"
        
        # Single core assessment
        if (( $(echo "$avg_cpu > 80" | bc -l) )); then
            echo -e "${YELLOW}⚠️  High CPU usage indicates single-core bottleneck${NC}"
        else
            echo -e "${GREEN}✅ CPU usage suggests room for more load${NC}"
        fi
    else
        echo "Unable to parse CPU data"
    fi
fi

# Memory usage
echo
echo -e "${BLUE}🧠 MEMORY USAGE${NC}"
echo "=============="
if ps -p $ENCRYPTOR_PID -o pid,vsz,rss,pmem --no-headers 2>/dev/null; then
    echo "(VSZ=Virtual Size, RSS=Resident Set Size, %MEM=Memory %)"
else
    echo "Process no longer running"
fi

# Error analysis
echo
echo -e "${BLUE}🚨 ERROR ANALYSIS${NC}"
echo "================"
if [[ -s /tmp/stress_errors.log ]]; then
    echo "Errors detected:"
    tail -5 /tmp/stress_errors.log
else
    echo -e "${GREEN}✅ No errors detected${NC}"
fi

# Latency estimation
if [[ $total_encrypted_lines -gt 0 ]]; then
    avg_latency=$(bc -l <<< "scale=3; $actual_duration * 1000 / $total_encrypted_lines")
    echo
    echo -e "${BLUE}⏱️  LATENCY ESTIMATION${NC}"
    echo "==================="
    echo "Average per-message time: ${avg_latency}ms"
    echo "(Including I/O, parsing, encryption, JSON output)"
fi

echo
echo -e "${GREEN}🎯 RECOMMENDATIONS${NC}"
echo "================="
if [[ $total_encrypted_lines -eq 0 ]]; then
    echo -e "${RED}❌ No messages were encrypted. Check socket connectivity.${NC}"
elif (( $(echo "$avg_cpu > 90" | bc -l) )); then
    echo -e "${YELLOW}🔥 Single core is saturated. Consider architecture changes for higher throughput.${NC}"
elif [[ $total_sent -gt $total_encrypted_lines ]]; then
    echo -e "${YELLOW}⚠️  Message loss detected. Increase socket buffer or reduce send rate.${NC}"
else
    echo -e "${GREEN}✅ System performing well. Try increasing load with more concurrent writers.${NC}"
fi

echo
echo -e "${BLUE}📁 Log files saved:${NC}"
echo "- Encrypted output: /tmp/stress_output.log"
echo "- Error log: /tmp/stress_errors.log"
echo "- CPU monitoring: /tmp/stress_cpu.log"