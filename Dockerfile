# Dockerfile
FROM golang:1.21-alpine AS builder

# Install git for dependency resolution
RUN apk add --no-cache git

# Set the working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create app directory
WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/main .

# Copy i18n files
COPY --from=builder /app/internal/i18n/locales ./internal/i18n/locales

# Expose port
EXPOSE 8080

# Command to run
CMD ["./main"]
