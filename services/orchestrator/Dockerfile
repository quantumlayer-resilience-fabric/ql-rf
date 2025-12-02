# Build stage
FROM golang:1.24rc2-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum for dependency caching
COPY go.mod go.sum ./
COPY services/orchestrator/go.mod services/orchestrator/go.sum* ./services/orchestrator/

# Download dependencies (allow toolchain auto-upgrade)
ENV GOTOOLCHAIN=auto
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /app/bin/orchestrator \
    ./services/orchestrator/cmd/orchestrator

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S app && \
    adduser -u 1000 -S app -G app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/orchestrator /app/orchestrator

# Copy OPA policies
COPY --from=builder /app/policy /app/policy

# Copy JSON schemas for validation
COPY --from=builder /app/schemas /app/schemas

# Change ownership
RUN chown -R app:app /app

# Switch to non-root user
USER app

# Expose port
EXPOSE 8083

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8083/health || exit 1

# Run the application
ENTRYPOINT ["/app/orchestrator"]
