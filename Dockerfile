# ComX-Bridge Dockerfile
# Multi-stage build for minimal image size

# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o /app/comx \
    ./cmd/comx

# Stage 2: Runtime
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1000 comx && \
    adduser -u 1000 -G comx -D comx

# Create necessary directories
RUN mkdir -p /etc/comx /var/log/comx /home/comx/plugins && \
    chown -R comx:comx /etc/comx /var/log/comx /home/comx

# Copy binary from builder
COPY --from=builder /app/comx /usr/local/bin/comx

# Copy default config
COPY configs/config.example.yaml /etc/comx/config.yaml

# Switch to non-root user
USER comx

# Set working directory
WORKDIR /home/comx

# Expose ports
# REST API
EXPOSE 8080
# gRPC
EXPOSE 9090
# WebSocket
EXPOSE 8081

# Environment variables
ENV COMX_CONFIG=/etc/comx/config.yaml
ENV COMX_LOG_LEVEL=info
ENV COMX_LOG_FORMAT=json

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
ENTRYPOINT ["comx"]
CMD ["start", "-c", "/etc/comx/config.yaml"]
