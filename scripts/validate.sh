#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="config-service"
LOCAL_PORT=8080
PASSED=0
FAILED=0

cleanup() {
  if [[ -n "${PF_PID:-}" ]]; then
    kill "$PF_PID" 2>/dev/null || true
    wait "$PF_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

echo "==> Checking pod status..."
kubectl get pods -n "$NAMESPACE"
echo ""

READY_PODS=$(kubectl get pods -n "$NAMESPACE" -l app=config-service -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}')
if [[ "$READY_PODS" == *"True"* ]]; then
  echo "PASS: config-service pod is Ready"
  ((PASSED++))
else
  echo "FAIL: config-service pod is not Ready"
  ((FAILED++))
fi

echo ""
echo "==> Starting port-forward..."
kubectl port-forward -n "$NAMESPACE" svc/config-service "$LOCAL_PORT":8080 &
PF_PID=$!
sleep 3

BASE_URL="http://localhost:${LOCAL_PORT}"

echo "==> Test 1: GET /ping"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/ping")
BODY=$(echo "$RESPONSE" | head -1)
STATUS=$(echo "$RESPONSE" | tail -1)
if [[ "$STATUS" == "200" && "$BODY" == "pong" ]]; then
  echo "PASS: /ping returned 200 with 'pong'"
  ((PASSED++))
else
  echo "FAIL: /ping returned status=$STATUS body=$BODY"
  ((FAILED++))
fi

echo ""
echo "==> Test 2: POST /configs (create)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/configs" \
  -H "Content-Type: application/json" \
  -d '{"id":"cfg_test","host":"localhost","port":8080,"app_name":"test-app","log_level":"INFO"}')
BODY=$(echo "$RESPONSE" | sed '$d')
STATUS=$(echo "$RESPONSE" | tail -1)
if [[ "$STATUS" == "201" ]]; then
  echo "PASS: POST /configs returned 201 (created)"
  ((PASSED++))
else
  echo "FAIL: POST /configs returned status=$STATUS body=$BODY"
  ((FAILED++))
fi

echo ""
echo "==> Test 3: GET /configs/cfg_test"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/configs/cfg_test")
BODY=$(echo "$RESPONSE" | sed '$d')
STATUS=$(echo "$RESPONSE" | tail -1)
if [[ "$STATUS" == "200" ]]; then
  echo "PASS: GET /configs/cfg_test returned 200"
  echo "      Response: $BODY"
  ((PASSED++))
else
  echo "FAIL: GET /configs/cfg_test returned status=$STATUS"
  ((FAILED++))
fi

echo ""
echo "==> Test 4: POST /configs (update same id)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/configs" \
  -H "Content-Type: application/json" \
  -d '{"id":"cfg_test","host":"127.0.0.1","port":9090,"app_name":"updated-app","log_level":"DEBUG"}')
BODY=$(echo "$RESPONSE" | sed '$d')
STATUS=$(echo "$RESPONSE" | tail -1)
if [[ "$STATUS" == "200" ]]; then
  echo "PASS: POST /configs (update) returned 200"
  ((PASSED++))
else
  echo "FAIL: POST /configs (update) returned status=$STATUS"
  ((FAILED++))
fi

echo ""
echo "==> Test 5: GET /configs/nonexistent"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/configs/nonexistent")
STATUS=$(echo "$RESPONSE" | tail -1)
if [[ "$STATUS" == "404" ]]; then
  echo "PASS: GET /configs/nonexistent returned 404"
  ((PASSED++))
else
  echo "FAIL: GET /configs/nonexistent returned status=$STATUS"
  ((FAILED++))
fi

echo ""
echo "==> Test 6: POST /configs (invalid body)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/configs" \
  -H "Content-Type: application/json" \
  -d '{"id":"","host":"","port":0,"app_name":"","log_level":"INVALID"}')
STATUS=$(echo "$RESPONSE" | tail -1)
if [[ "$STATUS" == "400" ]]; then
  echo "PASS: POST /configs (invalid) returned 400"
  ((PASSED++))
else
  echo "FAIL: POST /configs (invalid) returned status=$STATUS"
  ((FAILED++))
fi

echo ""
echo "==> Test 7: GET /healthz"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/healthz")
BODY=$(echo "$RESPONSE" | sed '$d')
STATUS=$(echo "$RESPONSE" | tail -1)
if [[ "$STATUS" == "200" ]]; then
  echo "PASS: /healthz returned 200"
  ((PASSED++))
else
  echo "FAIL: /healthz returned status=$STATUS"
  ((FAILED++))
fi

echo ""
echo "==> Test 8: GET /metrics"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/metrics")
if [[ "$STATUS" == "200" ]]; then
  echo "PASS: /metrics returned 200"
  ((PASSED++))
else
  echo "FAIL: /metrics returned status=$STATUS"
  ((FAILED++))
fi

echo ""
echo "================================"
echo "Results: ${PASSED} passed, ${FAILED} failed"
echo "================================"

if [[ "$FAILED" -gt 0 ]]; then
  exit 1
fi
