#!/usr/bin/env bash
# Validate the on-disk OLM bundle (Layer D).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SDK="${OPERATOR_SDK:-${ROOT}/bin/operator-sdk}"
CSV="${ROOT}/bundle/manifests/ip-rewrite-operator.clusterserviceversion.yaml"

if [[ ! -x "${SDK}" ]]; then
  echo "operator-sdk not found at ${SDK}"
  echo "Install: https://sdk.operatorframework.io/docs/installation/"
  echo "Or: make operator-sdk"
  exit 1
fi

echo "==> Structural checks"
test -f "${ROOT}/bundle/metadata/annotations.yaml"
test -f "${CSV}"
test -f "${ROOT}/bundle/manifests/iprewrite.reip.io_iprewriteconfigs.yaml"
test -f "${ROOT}/bundle/manifests/ip-rewrite-operator-webhook_v1_service.yaml"
grep -q 'relatedImages:' "${CSV}"
grep -q 'webhookdefinitions:' "${CSV}"
grep -q 'quay.io/mfagotto/ip-rewrite-operator:0.2.0' "${CSV}"
grep -q 'quay.io/mfagotto/ip-rewrite-operator:init-0.2.0' "${CSV}"
grep -q 'service.beta.openshift.io/serving-cert-secret-name' \
  "${ROOT}/bundle/manifests/ip-rewrite-operator-webhook_v1_service.yaml"
grep -q 'failurePolicy: Ignore' "${CSV}"
grep -q 'com.redhat.openshift.versions' "${ROOT}/bundle/metadata/annotations.yaml"
grep -q 'v4.22' "${ROOT}/bundle/metadata/annotations.yaml"
echo "structural OK"

echo "==> operator-sdk bundle validate"
"${SDK}" bundle validate "${ROOT}/bundle" --select-optional suite=operatorframework
echo "bundle validate OK"
