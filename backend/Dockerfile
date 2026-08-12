FROM golang:alpine AS builder

WORKDIR /app

# Install dependencies for CGO (SQLite requires CGO)
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=1 is required for go-sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build -o server main.go

# Use a lightweight alpine image for the final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/server .

# Expose port
EXPOSE 6001

# Command to run the executable
CMD ["./server"]
