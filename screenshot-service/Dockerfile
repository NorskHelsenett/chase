# Multi-stage build for crawler service
# 1. Go build stage
FROM golang:1.25-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
ARG SOURCE_DATE_EPOCH

ENV SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -trimpath \
    -buildvcs=false \
    -ldflags='-w -s -buildid=' \
    -o /app/crawler ./cmd/server

# 2. Runtime stage with Chrome/Chromium
FROM zenika/alpine-chrome:latest

# Install ca-certificates for HTTPS
USER root
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/crawler /app/crawler
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Set ownership (use existing chrome user from base image)
RUN chown -R chrome:chrome /app

# Switch to non-root user (chrome user from base image)
USER chrome

# Expose port
EXPOSE 11235

# Environment variables
ENV PORT=11235
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROME_PATH=/usr/lib/chromium/

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:11235/health || exit 1

ENTRYPOINT ["/app/crawler"]
