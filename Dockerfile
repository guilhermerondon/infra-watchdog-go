# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies.
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o uptime-go .

# Final stage
FROM alpine:latest

WORKDIR /root/

# Install CA certificates for HTTPS requests (since it monitors URLs)
RUN apk --no-cache add ca-certificates

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/uptime-go .

# Expose port (from main.go we know it listens on PORT, default 8080)
EXPOSE 8080

# Command to run the executable
CMD ["./uptime-go"]
