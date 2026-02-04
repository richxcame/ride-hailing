# Multi-stage Dockerfile for Go services
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Copy the replaced dependency so go mod download can see it
COPY third_party ./third_party

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build argument for service name
ARG SERVICE_NAME

# Build the service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/service ./cmd/${SERVICE_NAME}

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /home/appuser

# Copy the binary from builder
COPY --from=builder /app/service .

# Run as non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the service
CMD ["./service"]
