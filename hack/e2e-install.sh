#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "Applying operator manifests from config/default"
kubectl apply -k "${ROOT}/config/default"
echo "Waiting for Deployment..."
kubectl -n ip-rewrite-operator rollout status deployment/ip-rewrite-operator --timeout=180s
echo "Install OK"
