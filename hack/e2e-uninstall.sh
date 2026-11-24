#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "Deleting operator manifests (idempotent)"
kubectl delete -k "${ROOT}/config/default" --ignore-not-found
# Cluster-scoped leftovers sometimes linger if ownership was wrong
kubectl delete mutatingwebhookconfiguration ip-rewrite-operator --ignore-not-found
kubectl delete scc ip-rewrite-operator --ignore-not-found 2>/dev/null || true
kubectl delete clusterrole ip-rewrite-operator-manager ip-rewrite-operator-scc --ignore-not-found
kubectl delete clusterrolebinding ip-rewrite-operator-manager ip-rewrite-operator-scc --ignore-not-found
kubectl delete crd iprewriteconfigs.iprewrite.windowsreip.io --ignore-not-found
echo "Uninstall attempted — verify with: kubectl get mwc,crd | grep ip-rewrite || true"
