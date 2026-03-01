# Build Stage
FROM hub.infra.plz.ac/library/golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/server

# Run Stage
FROM hub.infra.plz.ac/library/alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the binary
COPY --from=builder /app/main .

# Copy default config (will be overridden by env vars in production)
COPY --from=builder /app/config.yaml ./config.yaml

# Create directories for dnsmasq config and PXE assets
RUN mkdir -p /etc/dnsmasq.d /tftpboot

EXPOSE 8000

CMD ["./main"]
