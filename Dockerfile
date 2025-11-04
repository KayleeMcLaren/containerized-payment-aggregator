# --- Stage 1: Build the Go application ---
# Use the official Go image for compilation
FROM golang:latest AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to leverage Docker layer caching for dependencies
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application. CGO_ENABLED=0 creates a statically linked binary (no runtime dependencies)
# -ldflags "-s -w" removes debugging symbols and reduces binary size.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o payment-gateway-aggregator .


# --- Stage 2: Create the Final, Minimal Runtime Image ---
# Use a very small, secure base image for the final runtime
FROM alpine:latest

# Set the working directory
WORKDIR /root/

# Copy the static binary from the builder stage
COPY --from=builder /app/payment-gateway-aggregator .

# Application is running on port 8080 (as defined in main.go)
EXPOSE 8080

# Define the command to run the executable
CMD ["./payment-gateway-aggregator"]