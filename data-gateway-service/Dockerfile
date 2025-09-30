# Build stage
FROM golang:1.25.1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy source code
COPY . .

# Download dependencies and build
RUN go mod download && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main_chi.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/server .

# Copy any config files if needed
# COPY --from=builder /app/configs ./configs

# Default to port 8081 but configurable via ENV
EXPOSE 8081

CMD ["./server"]