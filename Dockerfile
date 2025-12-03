# Build stage
FROM golang:1.25.4-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o achemdb-server ./cmd/achemdb-server

# Runtime stage
FROM alpine:latest

# Install ca-certificates and wget for HTTPS support and health checks
RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN addgroup -g 1000 achemdb && \
    adduser -D -u 1000 -G achemdb achemdb

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/achemdb-server /usr/local/bin/achemdb-server

# Set ownership
RUN chown achemdb:achemdb /usr/local/bin/achemdb-server

# Switch to non-root user
USER achemdb

# Expose default port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Run the server
ENTRYPOINT ["/usr/local/bin/achemdb-server"]

