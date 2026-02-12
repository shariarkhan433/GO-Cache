# Stage 1: The Builder (Compiling the code)
# We use the official Go image to build the binary
FROM golang:1.23-alpine AS builder

# Set working directory inside the container
WORKDIR /app

# Copy your source code
COPY . .

# Build the binary
# -o server: output name
# CGO_ENABLED=0: ensures it is a purely static binary (no C libraries linked)
RUN go build -o server main.go aof.go

# Stage 2: The Runner (The final tiny image)
# We use 'alpine' (a tiny Linux distro, ~5MB)
FROM alpine:latest

WORKDIR /root/

# Copy ONLY the binary from the builder stage
COPY --from=builder /app/server .

# Expose the Redis port
EXPOSE 6379

# Command to run when container starts
CMD ["./server"]