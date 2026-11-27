#!/usr/bin/env bash
# Build and push an OLM file-based catalog containing the operator bundle.
# Requires: opm (host), QUAY_USER, QUAY_PASS
# Optional: TAG=0.2.0 QUAY_ORG=mfagotto ENGINE=podman|docker OPM_VERSION=v1.74.0
#
# Catalog runtime is ubi10/ubi-micro + a static upstream opm binary (same VA
# pattern as the manager). Do NOT FROM quay.io/operator-framework/opm — that
# image embeds an old Go toolchain and fails the Quay fixable gate.
#
# Default images share the existing quay.io/<org>/ip-rewrite-operator repository
# (robot accounts cannot create new Quay repos). Override BUNDLE_IMG/CATALOG_IMG
# if separate -bundle/-catalog repositories have been created.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TAG="${TAG:-0.2.0}"
QUAY_ORG="${QUAY_ORG:-mfagotto}"
BUNDLE_IMG="${BUNDLE_IMG:-quay.io/${QUAY_ORG}/ip-rewrite-operator:bundle-${TAG}}"
CATALOG_IMG="${CATALOG_IMG:-quay.io/${QUAY_ORG}/ip-rewrite-operator:catalog-${TAG}}"
# Static opm release (go1.26+); pin for reproducible catalog builds.
OPM_VERSION="${OPM_VERSION:-v1.74.0}"
# Same ubi-micro digest as Dockerfile (manager runtime).
UBI_MICRO_DIGEST="${UBI_MICRO_DIGEST:-sha256:2ad0f98cb13160c4773c945ef93f88b25831c7f27cd3780d72a7e78b210ea72b}"

if [[ -z "${QUAY_USER:-}" || -z "${QUAY_PASS:-}" ]]; then
  echo "ERROR: set QUAY_USER and QUAY_PASS" >&2
  exit 1
fi
if ! command -v opm >/dev/null 2>&1; then
  echo "ERROR: opm not found in PATH (install from operator-framework/operator-registry)" >&2
  exit 1
fi

ENGINE="${ENGINE:-}"
if [[ -z "${ENGINE}" ]]; then
  if command -v podman >/dev/null 2>&1; then
    ENGINE=podman
  else
    ENGINE=docker
  fi
fi

echo "==> Login quay.io"
echo "${QUAY_PASS}" | ${ENGINE} login -u "${QUAY_USER}" --password-stdin quay.io

echo "==> Ensure bundle image ${BUNDLE_IMG}"
if ! ${ENGINE} image exists "${BUNDLE_IMG}" 2>/dev/null; then
  if [[ -f "${ROOT}/bundle.Dockerfile" ]]; then
    echo "Building bundle from bundle.Dockerfile..."
    ${ENGINE} build -t "${BUNDLE_IMG}" -f "${ROOT}/bundle.Dockerfile" "${ROOT}"
    ${ENGINE} push "${BUNDLE_IMG}"
  else
    ${ENGINE} pull "${BUNDLE_IMG}"
  fi
fi

TMP=$(mktemp -d)
trap 'rm -rf "${TMP}"' EXIT
mkdir -p "${TMP}/catalog" "${TMP}/bin"

# OLM CatalogSource pods exec grpc_health_probe from PATH (no extract-utilities
# init for plain grpc images). Ship a static probe next to opm.
GRPC_HEALTH_PROBE_VERSION="${GRPC_HEALTH_PROBE_VERSION:-v0.4.37}"

echo "==> Fetch static opm ${OPM_VERSION} + grpc_health_probe ${GRPC_HEALTH_PROBE_VERSION}"
curl -fsSL -o "${TMP}/bin/opm" \
  "https://github.com/operator-framework/operator-registry/releases/download/${OPM_VERSION}/linux-amd64-opm"
curl -fsSL -o "${TMP}/bin/grpc_health_probe" \
  "https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64"
chmod 0755 "${TMP}/bin/opm" "${TMP}/bin/grpc_health_probe"
"${TMP}/bin/opm" version >/dev/null

echo "==> opm render ${BUNDLE_IMG}"
opm render "${BUNDLE_IMG}" > "${TMP}/catalog/ip-rewrite-operator.v${TAG}.yaml"
cat > "${TMP}/catalog/ip-rewrite-operator.yaml" <<EOF
---
schema: olm.package
name: ip-rewrite-operator
defaultChannel: alpha
---
schema: olm.channel
package: ip-rewrite-operator
name: alpha
entries:
  - name: ip-rewrite-operator.v${TAG}
EOF

opm validate "${TMP}/catalog"

cat > "${TMP}/Dockerfile" <<EOF
# Catalog: ubi-micro + static opm ${OPM_VERSION} + grpc_health_probe (Quay fixable gate).
FROM registry.access.redhat.com/ubi10/ubi-micro@${UBI_MICRO_DIGEST}
COPY bin/opm /bin/opm
COPY bin/grpc_health_probe /bin/grpc_health_probe
COPY catalog /configs
USER 1001
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs"]
EOF

echo "==> Build ${CATALOG_IMG}"
${ENGINE} build -t "${CATALOG_IMG}" -f "${TMP}/Dockerfile" "${TMP}"
${ENGINE} push "${CATALOG_IMG}"

NEW_DIGEST="$(${ENGINE} inspect --format '{{.Digest}}' "${CATALOG_IMG}" 2>/dev/null || true)"
if [[ -z "${NEW_DIGEST}" ]]; then
  NEW_DIGEST="$(skopeo inspect --format '{{.Digest}}' "docker://${CATALOG_IMG}" 2>/dev/null || echo unknown)"
fi
echo "Pushed ${CATALOG_IMG} (${NEW_DIGEST})"
echo "Use as CatalogSource.spec.image"
echo "Quay VA: https://quay.io/repository/${QUAY_ORG}/ip-rewrite-operator/manifest/${NEW_DIGEST}?tab=vulnerabilities&fixable=true"