# ==========================================
# 1. BUILD STAGE (Runs on GitHub Actions Runner)
# ==========================================
FROM golang:alpine AS builder

WORKDIR /app

# Install security certificates & timezone data
RUN apk --no-cache add ca-certificates tzdata

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build stripped, highly optimized Linux binary (no debug symbols, lightweight size)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server ./cmd/api

# ==========================================
# 2. RUNTIME STAGE (Minimal lightweight image ~25 MB)
# ==========================================
FROM alpine:latest

WORKDIR /app

# Add runtime dependencies
RUN apk --no-cache add tzdata ca-certificates

# Copy compiled binary and configuration
COPY --from=builder /app/server .
COPY --from=builder /app/pkg/config/ ./pkg/config/

EXPOSE 3000

CMD ["./server"]