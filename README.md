# ip-rewrite-operator

OpenShift Virtualization add-on that rewrites static guest IPv4 (and optional DNS) on PVC-backed VM disks **before boot**, via a mutating admission webhook and a libguestfs init container.

Designed for OpenShift 4.22 / OpenShift Virtualization: **OpenShift Service CA** for webhook TLS (no cert-manager), optional SCC management, fail-open webhook.

Architecture inspired by [Soteria IP Rewrite](https://raffaelespazzoli.github.io/soteria/architecture/ip-rewrite/) — see [NOTICE](NOTICE).

## Repo

- Local: nested under the windowsreip lab tree for convenience  
- Origin: `https://github.com/mfagotto/ip-rewrite-operator.git`

## VM API (`iprewrite.windowsreip.io`)

| Kind | Key | Example |
|------|-----|---------|
| Label | `iprewrite.windowsreip.io/enabled` | `"true"` on `spec.template.metadata.labels` |
| Annotation | `iprewrite.windowsreip.io/<iface>-ip` | `"10.0.2.100/24;10.0.2.1"` |
| Annotation | `iprewrite.windowsreip.io/dns` | `"10.0.2.10,10.0.2.11"` |

Put labels/annotations on the **VM pod template** (`spec.template.metadata`) so KubeVirt copies them onto the virt-launcher pod.

## Cluster config

```yaml
apiVersion: iprewrite.windowsreip.io/v1alpha1
kind: IPRewriteConfig
metadata:
  name: cluster
spec:
  targetNamespaces: [vm-net-demo]
  requestKVMDevice: true
  manageSCC: true
```

## Develop

```bash
make test
make build
# images (when ready to push):
make docker-build
make docker-build-init
```

## Deploy (kustomize)

```bash
# After building/pushing images, edit config/manager/manager.yaml image refs if needed
oc apply -k config/default
```

Uninstall: `make undeploy` or `oc delete -k config/default`.

## Tests

| Layer | Command / location |
|-------|-------------------|
| A unit | `make test` |
| B fixtures | `hack/run-fixture-tests.sh` (WIP) |
| C e2e | `hack/e2e-install.sh` / `e2e-run.sh` / `e2e-uninstall.sh` (WIP) |

Repeatable install → test → uninstall on one OpenShift cluster is supported; only one operator install at a time (cluster-scoped MWC/CRD/SCC).

## Limitations (v1)

IPv4 only; RHEL 7–10 and Windows Server 2016–2025 / Windows 11; PVC disks only; no hostname rewrite; no BitLocker; skip on live migration; fail-open if webhook down.
