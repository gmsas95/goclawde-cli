# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy web UI
RUN mkdir -p bin/web && cp web/index.html bin/web/

# Build Go binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.version=$(git describe --tags --always 2>/dev/null || echo 'dev') -s -w" \
    -o jimmy ./cmd/jimmy

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/jimmy /app/jimmy

# Copy web UI
COPY --from=builder /build/bin/web /app/web

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
ENTRYPOINT ["/app/jimmy"]
CMD ["-data", "/app/data"]
