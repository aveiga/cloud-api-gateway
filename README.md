# Cloud API Gateway

A high-performance, configurable API Gateway written in Go with Keycloak integration for RBAC (Role-Based Access Control).

## Features

- **YAML-based Configuration**: Dynamic route configuration with regex pattern matching
- **Keycloak Integration**: Token introspection for authentication and authorization
- **RBAC Support**: Role-based access control with AND/OR logic
- **Token Caching**: Configurable token cache to reduce Keycloak load
- **Connection Pooling**: Efficient connection reuse for upstream services
- **Path Rewriting**: Strip prefixes before forwarding to upstream services
- **Graceful Shutdown**: Clean shutdown handling for production deployments

## Project Structure

```
cloud-api-gateway/
├── cmd/gateway/main.go           # Entry point, config loading, server startup
├── internal/
│   ├── config/config.go          # YAML config structs and loader
│   ├── auth/keycloak.go          # Keycloak introspection client
│   ├── middleware/
│   │   ├── auth.go               # JWT extraction and validation middleware
│   │   └── rbac.go               # Role-based access control middleware
│   ├── proxy/proxy.go            # Reverse proxy with connection pooling
│   └── router/router.go          # Regex-based route matching
├── config.example.yaml           # Example configuration
└── go.mod
```

## Building

### Local Build

```bash
go build ./cmd/gateway
```

### Docker Build

```bash
# Build for current platform
docker build -t cloud-api-gateway .

# Build for specific platform (e.g., Apple M-series)
docker build --platform linux/arm64 -t cloud-api-gateway .

# Build for multiple platforms
docker buildx build --platform linux/amd64,linux/arm64 -t cloud-api-gateway .
```

## Running

### Local Execution

```bash
# Using command line flag
./gateway -config config.yaml

# Using environment variable
CONFIG_PATH=config.yaml ./gateway
```

### Docker Execution

```bash
# Using Docker image from GitHub Container Registry
docker run -p 4010:4010 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  ghcr.io/aveiga/cloud-api-gateway:latest

# Or build and run locally
docker run -p 4010:4010 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  cloud-api-gateway
```

**Note**: Replace `aveiga/cloud-api-gateway` with your GitHub username/organization and repository name.

## Configuration

See `config.example.yaml` for a complete example configuration file.

### Key Configuration Options

- **Server**: Port, timeouts, and HTTP server settings
- **Keycloak**: Introspection URL, client credentials, and timeout
- **Cache**: Token caching settings (enabled/disabled, TTL)
- **Routes**: Route definitions with path patterns, methods, upstream URLs, and role requirements

### Environment Variable Substitution

Configuration supports environment variable substitution:

- `${VAR_NAME}` - Replaced with environment variable value
- `${VAR_NAME:-default}` - Uses default value if environment variable is not set

Example:

```yaml
client_secret: "${KEYCLOAK_CLIENT_SECRET}"
```

## Request Flow

```
Request → Router Match → Auth Middleware → RBAC Check → Reverse Proxy → Upstream
                ↓              ↓                ↓
            404 if no      401 if token      403 if
            route match    invalid           roles missing
```

## Dependencies

- `gopkg.in/yaml.v3` - YAML parsing (only external dependency)
- All other functionality uses Go standard library

## License

See LICENSE file for details.
