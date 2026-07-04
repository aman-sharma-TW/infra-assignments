#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="$SCRIPT_DIR/../terraform"
PASSWORD_FILE="$TERRAFORM_DIR/.db-password"

# Generate a random password if one doesn't exist yet for this cluster lifecycle
if [[ ! -f "$PASSWORD_FILE" ]]; then
  openssl rand -base64 24 > "$PASSWORD_FILE"
  chmod 600 "$PASSWORD_FILE"
fi

DB_PASSWORD=$(cat "$PASSWORD_FILE")

echo "==> Initializing Terraform..."
terraform -chdir="$TERRAFORM_DIR" init -input=false

echo "==> Applying Terraform configuration..."
terraform -chdir="$TERRAFORM_DIR" apply -auto-approve -input=false \
  -var="db_password=${DB_PASSWORD}"

echo "==> Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgresql \
  -n config-service --timeout=120s

echo "==> Waiting for config-service to be ready..."
kubectl wait --for=condition=ready pod -l app=config-service \
  -n config-service --timeout=120s

echo "==> Deployment complete!"
kubectl get pods -n config-service
