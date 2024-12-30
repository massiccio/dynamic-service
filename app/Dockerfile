# Dockerfile
FROM golang:1.23 AS builder

WORKDIR /app

# Copy the Go modules files and download dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the application code
COPY . .

# Build the application
RUN go build -v -o main .

# Use a minimal image for deployment
FROM gcr.io/distroless/base-debian12

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Expose port 8080
EXPOSE 18080

# Run the application
CMD ["./main"]
