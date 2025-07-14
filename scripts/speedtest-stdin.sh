#!/bin/bash

# Syslog Encryptor STDIN Performance Test
# Generates test data and measures encryption throughput

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Configuration
INPUT_FILE="/tmp/speedtest_input.txt"
OUTPUT_FILE="/tmp/speedtest_output.txt"
NUM_LINES=1000000
LINE_LENGTH=1024

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Syslog Encryptor STDIN Speed Test${NC}"
echo "===================================="
echo "Lines to generate: $NUM_LINES"
echo "Line length: $LINE_LENGTH bytes"
echo "Total test data: $(( NUM_LINES * LINE_LENGTH / 1024 / 1024 )) MB"
echo

# No cleanup - keep files for analysis

# Check if binary exists
if [[ ! -f "$PROJECT_DIR/syslog-encryptor" ]]; then
    echo -e "${RED}❌ syslog-encryptor binary not found. Run 'make build' first.${NC}"
    exit 1
fi

# Generate keys if they don't exist
echo -e "${BLUE}🔑 Setting up encryption keys...${NC}"
cd "$PROJECT_DIR"
eval "$(./scripts/generate-keys.sh | grep '^export')"

# Generate test input file
echo -e "${BLUE}📝 Generating test input file...${NC}"

# Generate a single line template (much faster than per-line generation)
template_line=$(printf '%*s' $LINE_LENGTH '' | tr ' ' 'A')

# Use 'yes' command - extremely fast way to generate repeated lines
echo "Generating $NUM_LINES lines of $LINE_LENGTH bytes each..."
yes "$template_line" | head -n $NUM_LINES > "$INPUT_FILE"

echo -e "${GREEN}✅ Test input file created: $(wc -l < "$INPUT_FILE") lines${NC}"

# Verify file size
actual_lines=$(wc -l < "$INPUT_FILE")
file_size=$(stat -c%s "$INPUT_FILE")
echo "Input file size: $(( file_size / 1024 / 1024 )) MB"

if [[ $actual_lines -ne $NUM_LINES ]]; then
    echo -e "${RED}❌ Expected $NUM_LINES lines, got $actual_lines${NC}"
    exit 1
fi

# Run speed test
echo
echo -e "${BLUE}⚡ Running speed test...${NC}"
echo "Command: cat \"$INPUT_FILE\" | STDIN_MODE=1 ./syslog-encryptor"
echo

# Record start time
start_time=$(date +%s.%N)

# Run the test
cat "$INPUT_FILE" | STDIN_MODE=1 ./syslog-encryptor > "$OUTPUT_FILE"

# Record end time
end_time=$(date +%s.%N)

# Calculate metrics
elapsed_time=$(echo "$end_time - $start_time" | bc -l)
output_lines=$(wc -l < "$OUTPUT_FILE")
lines_per_second=$(echo "scale=2; $output_lines / $elapsed_time" | bc -l)
mb_per_second=$(echo "scale=2; $file_size / $elapsed_time / 1024 / 1024" | bc -l)

# Display results
echo
echo -e "${GREEN}📊 SPEED TEST RESULTS${NC}"
echo "====================="
echo "Input lines: $actual_lines"
echo "Output lines: $output_lines"
echo "Processing time: ${elapsed_time}s"
echo "Lines per second: $lines_per_second"
echo "Throughput: ${mb_per_second} MB/s"
echo "Success rate: $(echo "scale=1; $output_lines * 100 / $actual_lines" | bc -l)%"

# Verify output format
echo
echo -e "${BLUE}🔍 OUTPUT VERIFICATION${NC}"
echo "===================="

# Check if output contains valid JSON
valid_json=0
sample_lines=100
head -n $sample_lines "$OUTPUT_FILE" | while IFS= read -r line; do
    if echo "$line" | jq -e 'has("t") and has("n") and has("m") and has("k")' > /dev/null 2>&1; then
        valid_json=$((valid_json + 1))
    fi
done

if [[ $valid_json -eq $sample_lines ]]; then
    echo -e "${GREEN}✅ All sampled output lines are valid encrypted JSON${NC}"
else
    echo -e "${YELLOW}⚠️  Some output lines may not be valid JSON${NC}"
fi

# Show sample output
echo
echo "Sample encrypted output:"
head -n 3 "$OUTPUT_FILE"

# Performance analysis
echo
echo -e "${BLUE}📈 PERFORMANCE ANALYSIS${NC}"
echo "======================"

avg_line_time=$(echo "scale=6; $elapsed_time / $output_lines * 1000" | bc -l)
echo "Average time per line: ${avg_line_time}ms"

# Performance tiers
if (( $(echo "$lines_per_second > 100000" | bc -l) )); then
    echo -e "${GREEN}🚀 Excellent performance (>100K lines/sec)${NC}"
elif (( $(echo "$lines_per_second > 50000" | bc -l) )); then
    echo -e "${GREEN}✅ Good performance (>50K lines/sec)${NC}"
elif (( $(echo "$lines_per_second > 10000" | bc -l) )); then
    echo -e "${YELLOW}⚠️  Moderate performance (>10K lines/sec)${NC}"
else
    echo -e "${RED}🐌 Low performance (<10K lines/sec)${NC}"
fi

echo
echo -e "${BLUE}📁 Files preserved:${NC}"
echo "- Input file: $INPUT_FILE ($(du -h "$INPUT_FILE" | cut -f1))"
echo "- Output file: $OUTPUT_FILE ($(du -h "$OUTPUT_FILE" | cut -f1))"
echo "- Files are kept for further analysis"
