# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o wosbTrade ./cmd/bot

# Runtime stage
FROM alpine:latest

# Install runtime dependencies including Node.js for Claude Code
RUN apk add --no-cache ca-certificates sqlite-libs tzdata nodejs npm

# Install Claude Code CLI globally
RUN npm install -g @anthropic-ai/claude-code

# Create non-root user
RUN addgroup -g 1000 botuser && \
    adduser -D -u 1000 -G botuser botuser

# Create data directory and Claude config directory
RUN mkdir -p /data/images /home/botuser/.config && \
    chown -R botuser:botuser /data /home/botuser/.config

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/wosbTrade .

# Switch to non-root user
USER botuser

# Set environment defaults
ENV DATABASE_PATH=/data/database.db \
    IMAGE_STORAGE_PATH=/data/images \
    LOG_LEVEL=info \
    CLAUDE_CODE_PATH=/usr/bin/claude

# Volume for persistent data (includes Claude config)
VOLUME ["/data", "/home/botuser/.config"]

# Run the bot
CMD ["./wosbTrade"]
