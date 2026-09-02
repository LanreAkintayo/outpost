# Multi-stage Dockerfile for Outpost

# Stage 1: Build stage with full Go toolchain
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for module downloading if needed
RUN apk add --no-cache git

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Compile optimized static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api cmd/api/main.go

# Stage 2: Minimal runtime image
FROM alpine:3.20

WORKDIR /app

# Install root CA certificates (required for outbound HTTPS webhook calls) and timezone data
RUN apk add --no-cache ca-certificates tzdata

# Run as non-root user for security
RUN addgroup -S outpost && adduser -S outpost -G outpost

COPY --from=builder /app/bin/api /app/api

USER outpost

EXPOSE 8080

ENTRYPOINT ["/app/api"]
