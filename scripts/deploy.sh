#!/usr/bin/env bash
set -euo pipefail

ENV="${1:-}"

case "$ENV" in
  dev)
    CONTAINER="toentoen_api_dev"
    TARGET_DIR="/home/ubuntu/envs/dev"
    DOMAIN="dev-api.toentoen.com"
    ;;
  staging)
    CONTAINER="toentoen_api_staging"
    TARGET_DIR="/home/ubuntu/envs/staging"
    DOMAIN="staging-api.toentoen.com"
    ;;
  preprod)
    CONTAINER="toentoen_api_preprod"
    TARGET_DIR="/home/ubuntu/envs/preprod"
    DOMAIN="preprod-api.toentoen.com"
    ;;
  prod)
    CONTAINER="toentoen_api"
    TARGET_DIR="/home/ubuntu"
    DOMAIN="api.toentoen.com"
    ;;
  *)
    echo "Penggunaan: ./deploy.sh [dev|staging|preprod|prod] [path_to_binary]"
    exit 1
    ;;
esac

SOURCE_BIN="${2:-/home/ubuntu/server_deploy}"

if [ ! -f "$SOURCE_BIN" ]; then
  echo "File binary $SOURCE_BIN tidak ditemukan!"
  exit 1
fi

echo "=========================================="
echo " Deploying ke $ENV ($CONTAINER)..."
echo "=========================================="
chmod +x "$SOURCE_BIN"

if [ "$ENV" = "prod" ]; then
  docker cp "$SOURCE_BIN" "$CONTAINER:/app/server"
else
  cp -f "$SOURCE_BIN" "$TARGET_DIR/server"
fi

docker restart "$CONTAINER"
sleep 2

echo "Menguji status HTTP https://$DOMAIN/..."
curl -s "https://$DOMAIN/"
echo ""
echo "Deployment ke $ENV ($DOMAIN) sukses!"
