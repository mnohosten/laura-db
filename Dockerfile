# Multi-stage build for LauraDB
# Stage 1: Build the Go binary
FROM golang:1.25.4-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o laura-server ./cmd/server

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS support
RUN apk --no-cache add ca-certificates

# Create non-root user for security
RUN addgroup -g 1000 laura && \
    adduser -D -u 1000 -G laura laura

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/laura-server .

# Create data directory and set ownership
RUN mkdir -p /data && chown -R laura:laura /app /data

# Switch to non-root user
USER laura

# Expose default port
EXPOSE 8080

# Set default environment variables for disk storage configuration
ENV DATA_DIR=/data \
    BUFFER_SIZE=1000 \
    DOC_CACHE=1000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/_health || exit 1

# Volume for persistent data storage
# Mount this to preserve database files across container restarts
VOLUME ["/data"]

# Run the server with disk storage configuration
ENTRYPOINT ["/app/laura-server"]
CMD ["-host", "0.0.0.0", "-port", "8080", "-data-dir", "/data", "-buffer-size", "1000", "-doc-cache", "1000"]
