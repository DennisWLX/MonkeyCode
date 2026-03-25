# ===========================================
# MonkeyCode Runner Docker Image
# ===========================================

# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install git for git operations
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build runner
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w" \
    -o runner \
    ./cmd/runner

# ===========================================
# Runtime stage
# ===========================================

FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    curl \
    ca-certificates \
    git \
    tzdata \
    && addgroup -S runner \
    && adduser -S runner -G runner

# Set non-root user
USER runner

# Copy runner binary from builder
COPY --from=builder /build/runner /usr/local/bin/runner

# Verify installation
RUN chmod +x /usr/local/bin/runner

# Create workspace directory
RUN mkdir -p /workspace /app/data && \
    chown -R runner:runner /workspace /app/data

WORKDIR /workspace

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command
CMD ["runner"]
