# ===================================
# Stage 1: Build Frontend
# ===================================
FROM node:18-alpine AS frontend-builder

WORKDIR /app/web

# Copy package files
COPY web/package*.json ./

# Install dependencies
RUN npm ci --legacy-peer-deps

# Copy frontend source
COPY web/ ./

# Build frontend
RUN npm run build

# ===================================
# Stage 2: Build Backend
# ===================================
FROM golang:1.23-alpine AS backend-builder

WORKDIR /app

# Install build dependencies (including gcc for CGO/SQLite)
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build backend (CGO_ENABLED=1 for SQLite support)
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o registry-sync-server \
    ./cmd/server

# ===================================
# Stage 3: Final Runtime Image
# ===================================
FROM alpine:latest

LABEL maintainer="zunshen"
LABEL description="Registry Sync - Docker Image Synchronization Tool"

# Install runtime dependencies (including libgcc for CGO-built binaries)
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    libgcc \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

# Copy backend binary from builder
COPY --from=backend-builder /app/registry-sync-server .

# Copy frontend build from builder
COPY --from=frontend-builder /app/web/build ./web/build

# Create data directory for database
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

# Run as non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser && \
    chown -R appuser:appgroup /app

USER appuser

# Set environment variables
ENV GIN_MODE=release

# Start server
CMD ["./registry-sync-server", "--port", "8080", "--db", "/app/data/registry-sync.db"]
