#!/usr/bin/env bash
# Post-OLM OpenShift wiring that cannot live in a registry+v1 bundle:
#   - Ensure operator Deployment uses OLM webhook cert mount
#   - virt-launcher SCC
#   - sample IPRewriteConfig
#
# OLM owns the MutatingWebhookConfiguration (service + caBundle). Do not point it
# at the Service-CA webhook Service — that fights CSV reconcile. The CSV/Deployment
# must serve TLS from /tmp/k8s-webhook-server/serving-certs (OLM-injected).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NS="${OPERATOR_NS:-ip-rewrite-operator}"

echo "==> Locate OLM MutatingWebhookConfiguration"
MWC=$(oc get mutatingwebhookconfiguration \
  -l olm.webhook-description-generate-name=ip-rewrite.reip.io \
  -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
if [[ -z "${MWC}" ]]; then
  MWC=$(oc get mutatingwebhookconfiguration -o name 2>/dev/null \
    | grep 'ip-rewrite.reip.io' | head -1 | sed 's|.*/||' || true)
fi
if [[ -z "${MWC}" ]]; then
  echo "ERROR: no MutatingWebhookConfiguration for ip-rewrite.reip.io" >&2
  exit 1
fi
echo "    MWC=${MWC}"

echo "==> Align Deployment webhook cert-dir with OLM mount"
# Idempotent: OLM may recreate pods; patch live Deployment after CSV is Succeeded.
ARGS=$(oc -n "${NS}" get deploy ip-rewrite-operator -o jsonpath='{.spec.template.spec.containers[0].args}' 2>/dev/null || echo '')
if [[ "${ARGS}" != *"/tmp/k8s-webhook-server/serving-certs"* ]]; then
  # Read current args and swap cert-dir
  oc -n "${NS}" get deploy ip-rewrite-operator -o json | python3 -c '
import json,sys
d=json.load(sys.stdin)
args=d["spec"]["template"]["spec"]["containers"][0].get("args") or []
out=[]
for a in args:
  if a.startswith("--cert-dir="):
    out.append("--cert-dir=/tmp/k8s-webhook-server/serving-certs")
  else:
    out.append(a)
if not any(x.startswith("--cert-dir=") for x in out):
  out.append("--cert-dir=/tmp/k8s-webhook-server/serving-certs")
d["spec"]["template"]["spec"]["containers"][0]["args"]=out
json.dump({"spec":{"template":{"spec":{"containers":[{"name":d["spec"]["template"]["spec"]["containers"][0]["name"],"args":out}]}}}}, sys.stdout)
' | oc -n "${NS}" patch deploy ip-rewrite-operator --type=strategic --patch-file=/dev/stdin
  oc -n "${NS}" rollout status deploy/ip-rewrite-operator --timeout=120s
else
  echo "    cert-dir already OLM path"
fi

echo "==> Apply SCC"
oc apply -f "${ROOT}/config/openshift/scc.yaml"

echo "==> Apply IPRewriteConfig sample"
oc apply -f "${ROOT}/config/samples/iprewrite_v1alpha1_iprewriteconfig.yaml"

echo "==> Status"
oc -n "${NS}" get csv,deploy,pods 2>/dev/null || oc -n "${NS}" get deploy,pods
oc get iprewriteconfig cluster -o wide 2>/dev/null || true
oc get scc ip-rewrite-operator >/dev/null && echo "SCC ip-rewrite-operator OK"
oc get mutatingwebhookconfiguration "${MWC}" -o jsonpath='service={.webhooks[0].clientConfig.service.name}:{.webhooks[0].clientConfig.service.port} nsSelector={.webhooks[0].namespaceSelector}{"\n"}'
echo "DONE — olm-post-install"
