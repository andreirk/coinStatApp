FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the code
COPY . .

# Build the app
RUN CGO_ENABLED=0 GOOS=linux go build -o coinStatApp ./cmd/coinStatApp

FROM alpine:3.18

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/coinStatApp .

# Expose the app port
EXPOSE 8080

# Run the app
ENTRYPOINT ["/app/coinStatApp"]
