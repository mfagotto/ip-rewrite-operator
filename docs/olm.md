# OLM bundle (OpenShift 4.22)

Maintainer / packaging notes for the Operator Lifecycle Manager bundle.  
**Day-2 install steps:** [hand-install-0.2.0.md](hand-install-0.2.0.md) · **end-user overview:** [user-guide.md](user-guide.md).

| | |
|---|---|
| Package | `ip-rewrite-operator` |
| Channel | `alpha` |
| CSV | `ip-rewrite-operator.v0.2.0` |
| Install modes | `OwnNamespace`, `SingleNamespace`, **`AllNamespaces`** (webhook must see virt-launcher pods outside the operator namespace). `MultiNamespace` is not supported. |
| OpenShift versions | `com.redhat.openshift.versions: v4.22` |

## Published images

Bundle and catalog tags live on the same Quay repository as the operator (public):

| Image | Tag |
|-------|-----|
| Bundle | `quay.io/mfagotto/ip-rewrite-operator:bundle-0.2.0` |
| Catalog | `quay.io/mfagotto/ip-rewrite-operator:catalog-0.2.0` |
| Manager | `quay.io/mfagotto/ip-rewrite-operator:0.2.0` |
| Init | `quay.io/mfagotto/ip-rewrite-operator:init-0.2.0` |

## Bundle contents (registry+v1)

| Manifest | Purpose |
|----------|---------|
| CSV | Deployment, RBAC, `webhookdefinitions`, `relatedImages` |
| CRD | `IPRewriteConfig` (`iprewrite.reip.io`) |
| Service | Webhook Service (OLM also generates an admission `*-service`) |

`MutatingWebhookConfiguration` and `SecurityContextConstraints` are **not** valid registry+v1 bundle objects. OLM creates the MWC from CSV `webhookdefinitions` and mounts TLS at `/tmp/k8s-webhook-server/serving-certs`. After CSV Succeeded, [hack/olm-post-install.sh](../hack/olm-post-install.sh) applies SCC + sample `IPRewriteConfig` and aligns the Deployment `--cert-dir` with that mount.

Kustomize (`oc apply -k config/`) is the non-OLM path and uses OpenShift Service CA on the webhook Service instead of OLM webhook certs — see [user-guide.md](user-guide.md).

## Validate / build / publish

```bash
make bundle-validate
make bundle-build   # → BUNDLE_IMG (default :bundle-0.2.0)
export QUAY_USER=… QUAY_PASS=…   # push credentials (not needed for cluster pulls of public tags)
podman push "$BUNDLE_IMG"
./hack/catalog-build.sh   # needs opm; pushes :catalog-0.2.0
                          # runtime: ubi10/ubi-micro + static opm v1.74.0
```

After publishing a same-version catalog rebuild, bump the CatalogSource image digest (or wait for poll) so OLM reloads the CSV.
