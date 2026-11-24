#!/usr/bin/env bash
# Cluster e2e smoke: assumes images are pullable and oc/kubectl is logged in.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NS="${E2E_NS:-vm-net-demo}"

"${ROOT}/hack/e2e-install.sh"

echo "Checking MWC caBundle..."
CABUNDLE=$(kubectl get mutatingwebhookconfiguration ip-rewrite-operator \
  -o jsonpath='{.webhooks[0].clientConfig.caBundle}' || true)
if [[ -z "${CABUNDLE}" ]]; then
  echo "WARN: caBundle empty — Service CA injection may still be pending"
else
  echo "caBundle present ($(echo -n "$CABUNDLE" | wc -c) bytes)"
fi

echo "Checking webhook Service serving-cert secret..."
kubectl -n ip-rewrite-operator get secret ip-rewrite-operator-webhook-cert

echo "E2E smoke (install path) OK for namespace hint: ${NS}"
echo "Next: label a disposable VM, stop/start, verify init container named ip-rewrite on virt-launcher."
