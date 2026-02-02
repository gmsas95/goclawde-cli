# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git npm nodejs

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build web UI
RUN cd web && npm install && npm run build

# Build Go binary
RUN CGO_ENABLED=1 GOOS=linux go build -tags=embed \
    -ldflags "-X main.version=$(git describe --tags --always) -s -w" \
    -o nanobot ./cmd/nanobot

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/nanobot /app/nanobot

# Create data directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Volume for data persistence
VOLUME ["/app/data"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Run the binary
ENTRYPOINT ["/app/nanobot"]
CMD ["-data", "/app/data"]
