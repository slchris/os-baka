#!/usr/bin/env bash
set -e

# Configuration — override with environment variables
REMOTE_USER="${REMOTE_USER:-user}"
REMOTE_HOST="${REMOTE_HOST:-10.31.0.3}"
REMOTE_DIR="${REMOTE_DIR:-/home/$REMOTE_USER/os-baka}"
ARCH="${ARCH:-linux/amd64}"

if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "Usage: REMOTE_HOST=192.168.1.100 REMOTE_USER=deploy ./deploy.sh"
    echo ""
    echo "Environment variables:"
    echo "  REMOTE_USER  - SSH user (default: user)"
    echo "  REMOTE_HOST  - Target host (default: 10.31.0.3)"
    echo "  REMOTE_DIR   - Remote directory (default: /home/\$REMOTE_USER/os-baka)"
    echo "  ARCH         - Docker platform (default: linux/amd64)"
    exit 0
fi

echo "==== OS Baka Deployment Script ===="
echo "Target: $REMOTE_USER@$REMOTE_HOST"
echo "Architecture: $ARCH"

# 1. Build
echo "[1/4] Building images locally..."
export DOCKER_DEFAULT_PLATFORM=$ARCH
docker compose -f docker/docker-compose.prod.yml build --no-cache

# Pull base images ensuring platform
echo "Pulling base images for $ARCH..."
docker pull --platform $ARCH hub.infra.plz.ac/library/postgres:15-alpine

# 2. Save
echo "[2/4] Saving and compressing images..."
docker save \
    osbaka-backend:latest \
    osbaka-frontend:latest \
    osbaka-pxe:latest \
    hub.infra.plz.ac/library/postgres:15-alpine \
    | gzip > deploy_images.tar.gz

echo "Size: $(du -h deploy_images.tar.gz | cut -f1)"

# 3. Transfer
echo "[3/4] Transferring contents to remote..."
ssh $REMOTE_USER@$REMOTE_HOST "mkdir -p $REMOTE_DIR"
scp deploy_images.tar.gz $REMOTE_USER@$REMOTE_HOST:/tmp/
scp docker/docker-compose.prod.yml $REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/docker-compose.yml

# 4. Remote Execution
echo "[4/4] Executing remote setup..."
ssh $REMOTE_USER@$REMOTE_HOST << EOF
    set -e
    cd $REMOTE_DIR
    
    echo "Checking for port conflicts (80, 8000, 5432, 67, 69)..."
    for port in 80 8000 5432 67 69; do
        if ss -tuln | grep -q ":$port "; then
            echo "WARNING: Port $port is already in use!"
        else
            echo "Port $port is free."
        fi
    done

    echo "Loading images (this may take a moment)..."
    gunzip -c /tmp/deploy_images.tar.gz | docker load
    
    echo "Starting services..."
    docker compose down --remove-orphans || true
    # Force recreate to ensure fresh environment
    docker compose up -d --force-recreate

    echo "Deployment Complete!"
    docker compose ps
    
    echo ""
    echo "=== PXE Service Status ==="
    echo "DHCP Server: Port 67/UDP"
    echo "TFTP Server: Port 69/UDP"
    echo "Make sure these ports are accessible on your network."
EOF

# Cleanup
rm deploy_images.tar.gz
echo "Done."
