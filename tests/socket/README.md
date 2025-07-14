# Syslog Socket Test

This test verifies that the syslog-encryptor properly handles Unix domain socket communication with Go's standard `log/syslog` library.

## Components

- **syslog-encryptor**: SOCK_DGRAM Unix socket server that encrypts incoming syslog messages
- **syslog-generator**: Go program using `log/syslog` to send test messages

## Files

- `syslog-generator.go`: Test program that generates syslog messages
- `Dockerfile`: Builds the syslog-generator container
- `docker-compose.yaml`: Orchestrates the test environment
- `README.md`: This file

## Test Scenario

1. syslog-encryptor starts and creates `/syslog/test.sock` Unix domain socket (SOCK_DGRAM)
2. syslog-generator waits for socket, then symlinks it to `/dev/log`
3. Generator connects using Go's `log/syslog` package
4. Generator sends 5 test messages with 200ms delay
5. Encryptor receives, encrypts, and outputs JSON to stdout
6. Test validates SOCK_DGRAM compatibility and newline handling

## Running the Test

### Prerequisites

Ensure the main syslog-encryptor is built:
```bash
cd ../..
go build -o syslog-encryptor
```

### Run Test

```bash
# Run the full test
docker compose up

# Run with logs
docker compose up --build

# Clean up
docker compose down
```

### Expected Output

You should see:
1. syslog-encryptor startup: "Unix syslog encryptor listening on /syslog/test.sock (SOCK_DGRAM)"
2. syslog-generator waiting: "Socket found, linking and starting generator..."
3. Generator statistics: "Sent 5 messages in 1.002s, Rate: 4.99 messages/second"
4. Five encrypted JSON messages output to stdout

Example encrypted output:
```json
{"t":"2025-07-14T15:10:52.188807679Z","n":"8Jg+FuT6mM6oQ5b8","m":"7Onh5Mfp0JaBEh4+scwqyjd...","k":"f17cdf9b9d2430cec9f4793bdc12101ee50475cb0944564d27b0c3e8c1dafa5e"}
```

## Test Parameters

Default generator settings:
- **Messages**: 5
- **Size**: 64 bytes each
- **Tag**: "socket-test"  
- **Delay**: 200ms between messages

Modify in `docker-compose.yaml` command section to adjust test parameters.

## Validation

A successful test shows:
- ✅ SOCK_DGRAM socket creation
- ✅ Go syslog library compatibility
- ✅ Message encryption and JSON output
- ✅ No connection errors
- ✅ All messages processed

## Key Differences from SOCK_STREAM

This test validates the migration from SOCK_STREAM to SOCK_DGRAM:
- **Connection model**: Connectionless vs connection-oriented
- **Message boundaries**: Packet-based vs stream-based
- **Delimiter handling**: Newline stripping vs null termination
- **Compatibility**: Modern syslogd standard vs legacy approach

## Troubleshooting

**Generator fails to connect**:
- Check if encryptor is running and listening
- Verify `/dev/log` socket exists and has correct permissions

**No encrypted output**:
- Check environment variables are set correctly
- Verify key pair compatibility

**Permission errors**:
- Socket permissions should be 0666
- Containers should have access to shared volume