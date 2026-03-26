# ===========================================
# MonkeyCode Runner Docker Image
# ===========================================

# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install git for git operations
RUN apk add --no-cache git ca-certificates

# Copy go mod files and vendor directory
COPY go.mod go.sum ./
COPY vendor/ ./vendor/

# Copy source code
COPY . .

# Build runner using vendor mode
RUN go build -mod=vendor \
    -ldflags "-s -w" \
    -o runner \
    ./cmd/runner

# Remove vendor directory from build stage (not needed in final image)
RUN rm -rf /build/vendor

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
    && addgroup -S docker \
    && addgroup -S runner \
    && adduser -S runner -G runner \
    && addgroup runner docker

# Copy runner binary from builder
COPY --from=builder /build/runner /usr/local/bin/runner

# Verify installation (must be before USER change)
RUN chmod +x /usr/local/bin/runner

# Create workspace directory (must be before USER change)
RUN mkdir -p /workspace /app/data && \
    chown -R runner:runner /workspace /app/data

# Set non-root user
USER runner

WORKDIR /workspace

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command
CMD ["runner"]
