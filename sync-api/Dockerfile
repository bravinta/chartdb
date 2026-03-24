FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependencies list first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o sync-api main.go

# Create a lightweight image
FROM alpine:latest

WORKDIR /app

# Install CA certificates for HTTPS requests if needed
RUN apk --no-cache add ca-certificates

# Copy the compiled binary from the builder stage
COPY --from=builder /app/sync-api .

# Expose the API port
EXPOSE 3001

# Command to run the application
CMD ["./sync-api"]
