#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="config-service"
IMAGE_NAME="config-service:latest"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "==> Checking prerequisites..."
for cmd in docker kind kubectl helm terraform; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "ERROR: $cmd is not installed"
    exit 1
  fi
done

echo "==> Creating Kind cluster '${CLUSTER_NAME}'..."
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "    Cluster already exists, skipping creation"
else
  kind create cluster --name "$CLUSTER_NAME" --wait 60s
fi

echo "==> Building Docker image..."
docker build -t "$IMAGE_NAME" "$PROJECT_DIR"

echo "==> Loading image into Kind cluster..."
kind load docker-image "$IMAGE_NAME" --name "$CLUSTER_NAME"

echo "==> Running deployment..."
"$SCRIPT_DIR/deploy.sh"

echo ""
echo "==> Setup complete!"
echo "    Run './scripts/validate.sh' to verify the deployment."
echo "    Run 'kubectl port-forward -n config-service svc/config-service 8080:8080' to access the service."
