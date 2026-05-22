# Build Stage
FROM hub.infra.plz.ac/library/golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/server

# Run Stage
FROM hub.infra.plz.ac/library/alpine:latest

# ipmitool is required for BMC power management (manual /nodes/:id/power
# action and the auto power-cycle that follows a rebuild). Without it both
# code paths fail at exec time with "executable file not found".
RUN apk add --no-cache ca-certificates tzdata ipmitool

WORKDIR /app

# Copy the binary
COPY --from=builder /app/main .

# Create directories for dnsmasq config and PXE assets
RUN mkdir -p /etc/dnsmasq.d /tftpboot

EXPOSE 8000

CMD ["./main"]
