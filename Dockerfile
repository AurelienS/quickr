# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy only the files needed for go mod download
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o quickr

# Final stage
FROM alpine:3.18

WORKDIR /app

# Install runtime dependencies for SQLite
RUN apk add --no-cache sqlite

# Create a non-root user
RUN adduser -D -h /app quickr
USER quickr

# Copy the binary from builder
COPY --from=builder /app/quickr .

# Create data directory owned by quickr user
RUN mkdir -p /app/data && chown -R quickr:quickr /app/data

# Expose the port
EXPOSE 8080

# Set the data directory as a volume
VOLUME ["/app/data"]

# Run the application
CMD ["./quickr"]