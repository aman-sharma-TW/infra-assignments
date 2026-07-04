#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="config-service"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="$SCRIPT_DIR/../terraform"
PASSWORD_FILE="$TERRAFORM_DIR/.db-password"

echo "==> Destroying Terraform resources..."
if [[ -d "$TERRAFORM_DIR/.terraform" ]]; then
  if [[ -f "$PASSWORD_FILE" ]]; then
    DB_PASSWORD=$(cat "$PASSWORD_FILE")
    terraform -chdir="$TERRAFORM_DIR" destroy -auto-approve -input=false \
      -var="db_password=${DB_PASSWORD}" 2>/dev/null || true
  else
    terraform -chdir="$TERRAFORM_DIR" destroy -auto-approve -input=false \
      -var="db_password=unused" 2>/dev/null || true
  fi
fi

echo "==> Cleaning up sensitive files..."
rm -f "$PASSWORD_FILE"
rm -f "$TERRAFORM_DIR/terraform.tfstate"
rm -f "$TERRAFORM_DIR/terraform.tfstate.backup"

echo "==> Deleting Kind cluster '${CLUSTER_NAME}'..."
kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true

echo "==> Teardown complete!"
