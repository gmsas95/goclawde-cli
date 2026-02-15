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
    -o goclawde ./cmd/goclawde

FROM alpine:latest

LABEL org.opencontainers.image.title="GoClawde"
LABEL org.opencontainers.image.description="Personal AI Assistant"
LABEL org.opencontainers.image.source="https://github.com/gmsas95/goclawde-cli"

RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

COPY --from=builder /build/goclawde /app/goclawde
COPY --from=builder /build/bin/web /app/web

RUN mkdir -p /app/data

EXPOSE 8080

VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/goclawde"]
CMD ["--data", "/app/data"]
