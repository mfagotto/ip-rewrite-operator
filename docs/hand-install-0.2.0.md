# Hand install — IP Rewrite Operator 0.2.0 (OLM)

Step-by-step OLM install on OpenShift 4.22. API group: **`iprewrite.reip.io`**.

Packaging metadata and how to build the bundle/catalog: [olm.md](olm.md).  
VM opt-in, verify, and limitations: [user-guide.md](user-guide.md).

## Prerequisites

- OpenShift 4.22 with OpenShift Virtualization
- Cluster admin (`oc` logged in)
- Network reachability to the public Quay tags listed in [olm.md](olm.md) (no pull secret required)

## 1. Remove a prior install (if any)

```bash
oc get csv,sub,catsrc -A | grep -i ip-rewrite || echo clean
oc get mutatingwebhookconfiguration -o name | grep -i rewrite || echo no-mwc
oc get crd iprewriteconfigs.iprewrite.windowsreip.io 2>/dev/null && echo "REMOVE old CRD" || true
```

```bash
oc delete -k config/ --ignore-not-found
oc delete crd iprewriteconfigs.iprewrite.windowsreip.io --ignore-not-found
# If an old OLM install remains, also delete its Subscription / CSV / CatalogSource in ip-rewrite-operator
```

## 2. CatalogSource + Subscription

```bash
oc apply -f examples/olm-subscription.yaml
```

Creates namespace `ip-rewrite-operator`, an **AllNamespaces** OperatorGroup, CatalogSource (`:catalog-0.2.0`), and Subscription (channel `alpha`).

```bash
oc -n ip-rewrite-operator get catalogsource ip-rewrite-operator
# CONNECTION STATE = READY
```

## 3. Wait for CSV Succeeded

```bash
oc -n ip-rewrite-operator get csv
# Expect: ip-rewrite-operator.v0.2.0 … Succeeded
oc -n ip-rewrite-operator get deploy,pods
# Deployment 2/2 Ready
```

## 4. Post-install wiring

Registry+v1 cannot ship SCC or finish webhook cert-dir alignment. Run:

```bash
./hack/olm-post-install.sh
```

That script (committed under `hack/`) applies:

- virt-launcher SCC (`config/openshift/scc.yaml`)
- sample `IPRewriteConfig/cluster` (edit `targetNamespaces` for your VM namespaces)
- Deployment `--cert-dir=/tmp/k8s-webhook-server/serving-certs` to match the OLM TLS mount

```bash
oc get iprewriteconfig cluster -o yaml
oc get scc ip-rewrite-operator
oc get mutatingwebhookconfiguration -l olm.webhook-description-generate-name=ip-rewrite.reip.io
```

## 5. Console

**Operators → Installed Operators → IP Rewrite Operator** (0.2.0).

Then annotate VMs per the [user guide](user-guide.md) (label `enabled=true` + `*-ip` on `spec.template.metadata`).
