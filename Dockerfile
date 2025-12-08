# Multi-architecture Dockerfile
# Supports: linux/amd64, linux/arm64 (Apple M-series, AWS Graviton)
# Build for current platform: docker build -t cloud-api-gateway .
# Build for multiple platforms: docker buildx build --platform linux/amd64,linux/arm64 -t cloud-api-gateway .

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for multi-architecture support
# TARGETARCH and TARGETOS are automatically set by Docker buildx
ARG TARGETARCH
ARG TARGETOS=linux

# Build the application
# CGO_ENABLED=0 creates a statically linked binary
# -ldflags="-w -s" strips debug information to reduce binary size
# Supports both amd64 and arm64 (Apple M-series) architectures
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s" \
    -o gateway \
    ./cmd/gateway

# Runtime stage
FROM alpine:latest

# Install CA certificates and wget for healthcheck
RUN apk --no-cache add ca-certificates tzdata wget

# Create non-root user for security
RUN addgroup -g 1000 gateway && \
    adduser -D -u 1000 -G gateway gateway

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/gateway /app/gateway

# Copy example config (users should mount their own config.yaml)
COPY --from=builder /build/config.example.yaml /app/config.example.yaml

# Change ownership to non-root user
RUN chown -R gateway:gateway /app

# Switch to non-root user
USER gateway

# Expose default port (configurable via config file)
EXPOSE 4010

# Health check (requires a /health route to be configured in config.yaml)
# Adjust the port number to match your server.port configuration
# Comment out if no health endpoint is configured
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:4010/health || exit 1

# Run the gateway
# Default config path can be overridden via CONFIG_PATH env var or -config flag
ENTRYPOINT ["/app/gateway"]
CMD ["-config", "/app/config.yaml"]

