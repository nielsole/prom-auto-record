# Stage 1: Build the Go application
FROM golang:1.21 AS build

# Set working directory
WORKDIR /app

# Install dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o prom-auto-record ./cmd/prom-auto-record

# Stage 2: Create a minimal runtime image
FROM alpine:3.18

# Set working directory
WORKDIR /app

# Copy binary from the build stage
COPY --from=build /app/prom-auto-record .

# Run the application
CMD ["./prom-auto-record"]
