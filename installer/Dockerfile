# ===========================================
# MonkeyCode Runner Installer Builder
# ===========================================

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build for multiple platforms
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o installer_linux_amd64 .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o installer_linux_arm64 .

# Final stage - create distributable packages
FROM alpine:3.19

WORKDIR /app

# Install curl for health checks
RUN apk add --no-cache curl ca-certificates

# Copy binaries from builder
COPY --from=builder /build/installer_linux_amd64 /app/installer
COPY --from=builder /build/installer_linux_arm64 /app/installer-arm64

# Create version file
RUN echo "1.0.0" > /app/VERSION

# Create entrypoint script
RUN echo '#!/bin/sh\necho "Please run this binary directly, not in Docker"\n/app/installer "$@"' > /entrypoint.sh && chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
