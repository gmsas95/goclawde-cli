# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p bin/web && cp web/index.html bin/web/ 2>/dev/null || mkdir -p bin/web

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.version=$(git describe --tags --always 2>/dev/null || echo 'dev') -s -w" \
    -o myrai ./cmd/myrai

# Production stage
FROM alpine:latest

LABEL org.opencontainers.image.title="Myrai (未来)"
LABEL org.opencontainers.image.description="Personal AI Assistant for Everyone"
LABEL org.opencontainers.image.source="https://github.com/gmsas95/myrai-cli"

RUN apk add --no-cache ca-certificates sqlite-libs tzdata

WORKDIR /app

# Create non-root user
RUN addgroup -g 1000 myrai && \
    adduser -u 1000 -G myrai -s /bin/sh -D myrai

COPY --from=builder /build/myrai /app/myrai
COPY --from=builder /build/bin/web /app/web

# Create data directory with proper permissions
RUN mkdir -p /app/data && chown -R myrai:myrai /app/data

USER myrai

EXPOSE 8080

VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/myrai"]
CMD ["server", "--data", "/app/data"]
