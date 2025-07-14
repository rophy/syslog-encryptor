FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY syslog-generator.go .

# Build the syslog generator
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o syslog-generator syslog-generator.go

FROM alpine:latest

# Install ca-certificates for any HTTPS requests (if needed)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/syslog-generator .

# Default command
CMD ["./syslog-generator", "-n", "10", "-s", "64", "-d", "100ms"]