# IP Rewrite Operator — User guide

Rewrite static guest IPv4 (and optional DNS) on OpenShift Virtualization VMs **before the guest boots**. A mutating webhook injects a libguestfs init container that edits the root disk offline. This is **not** CloudInit-based reIP.

**Version:** 0.2.0 (pre-production beta)

| | |
|---|---|
| Operator image | `quay.io/mfagotto/ip-rewrite-operator:0.2.0` |
| Init image | `quay.io/mfagotto/ip-rewrite-operator:init-0.2.0` |
| API group | `iprewrite.reip.io` |

Example fragments: [examples/vm-snippet.yaml](../examples/vm-snippet.yaml) (Linux and Windows).  
How it works: [architecture.md](architecture.md).

---

## 1. Prerequisites

- OpenShift **4.22** with OpenShift Virtualization (KubeVirt + CDI)
- Cluster rights to create namespaces, CRDs, ClusterRoles, SCCs, and MutatingWebhookConfigurations (or an admin who installs the operator for you)
- Network reachability to Quay for the images above. The published `mfagotto/ip-rewrite-operator` tags are **public** — no pull secret is required. Use a docker-registry pull secret only if you mirror to a **private** registry or lock the Quay repo down later.
- A namespace for workload VMs
- PVC-backed VM root disks
- Supported guests: **RHEL 7–10**, **Windows Server 2016–2025** / Windows 11  
  CentOS Stream guests are **not** supported

---

## 2. Install with manifests (kustomize)

Recommended when you manage the operator with GitOps or plain YAML.

```bash
oc apply -k config/
```

This installs the CRD, RBAC, operator Deployment, webhook, SCC, and a sample `IPRewriteConfig` into namespace `ip-rewrite-operator`.

Verify:

```bash
oc -n ip-rewrite-operator get deploy,pods
oc get mutatingwebhookconfiguration ip-rewrite-operator
oc get scc ip-rewrite-operator
oc get iprewriteconfig cluster
```

Expect the Deployment **Ready**, the MutatingWebhookConfiguration present, and (with `manageSCC: true`) the SCC present.

Uninstall:

```bash
oc delete -k config/
```

If you also installed via OLM, remove the Subscription / CSV first so the two paths do not fight.

---

## 3. Install with OLM

Use this path when you want the operator under **Operators → Installed Operators**.

1. Ensure no kustomize install is still running (`oc delete -k config/` if needed).
2. Apply the CatalogSource and Subscription:

   ```bash
   oc apply -f examples/olm-subscription.yaml
   ```

3. Wait until the catalog is ready and the CSV succeeds:

   ```bash
   oc -n ip-rewrite-operator get catalogsource ip-rewrite-operator
   oc -n ip-rewrite-operator get csv
   ```

4. Complete OpenShift wiring that is outside the OLM bundle (SCC, sample config, webhook cert path):

   ```bash
   ./hack/olm-post-install.sh
   ```

5. In the console: **Installed Operators** → **IP Rewrite Operator**.

Step-by-step OLM checklist: [hand-install-0.2.0.md](hand-install-0.2.0.md).  
Publishing bundle/catalog images is a **maintainer** task: [olm.md](olm.md).

---

## 4. Cluster configuration

The operator is configured with a cluster-scoped `IPRewriteConfig` (name `cluster`):

```yaml
apiVersion: iprewrite.reip.io/v1alpha1
kind: IPRewriteConfig
metadata:
  name: cluster
spec:
  targetNamespaces:
    - my-vms                    # namespaces where applied-stamping is allowed
  virtLauncherServiceAccountNames:
    - default
  initContainerImage: quay.io/mfagotto/ip-rewrite-operator:init-0.2.0
  initContainerPullPolicy: IfNotPresent
  requestKVMDevice: true
  manageSCC: true
  webhookFailurePolicy: Ignore
```

Apply or edit as needed:

```bash
oc apply -f config/samples/iprewrite_v1alpha1_iprewriteconfig.yaml
oc edit iprewriteconfig cluster
```

Confirm the webhook serving certificate and CA bundle:

```bash
oc -n ip-rewrite-operator get secret ip-rewrite-operator-webhook-cert
oc get mutatingwebhookconfiguration ip-rewrite-operator \
  -o jsonpath='{.webhooks[0].clientConfig.caBundle}' | wc -c
```

A non-zero CA bundle length means the Service CA injection path is populated.

---

## 5. Opt in a VirtualMachine

Put labels and annotations on **`spec.template.metadata`** so KubeVirt copies them onto the virt-launcher pod.

| Kind | Key | Example |
|------|-----|---------|
| **Label** (required) | `iprewrite.reip.io/enabled` | `"true"` |
| Annotation | `iprewrite.reip.io/<iface>-ip` | `"192.168.100.60/24;192.168.100.1"` |
| Annotation | `iprewrite.reip.io/dns` | `"8.8.8.8,1.1.1.1"` |
| Annotation | `iprewrite.reip.io/applied` | Fingerprint of desired IP/DNS (usually stamped automatically) |

Copy-paste fragments: [examples/vm-snippet.yaml](../examples/vm-snippet.yaml).

**The label is the opt-in.** IP annotations alone do not inject the init container.

### Choosing `<iface>` (important in 0.2.0)

The annotation key is `iprewrite.reip.io/<iface>-ip`. Meaning of `<iface>`:

| Guest | What to put in `<iface>` |
|-------|---------------------------|
| **RHEL / Linux** | The **real** guest device name (NetworkManager `interface-name` or ifcfg `DEVICE`). RHEL 8/9 often use `eth0`. **RHEL 10** commonly uses `enp1s0`. A wrong name means the init finds no config and fails. Confirm with `ip -br link` on the guest (or on your golden image) before annotating. |
| **Windows** (typical single NIC) | A **logical** name in the key only. The handler selects the real Tcpip adapter; it does not require the Windows connection to be named `eth0`. Using `eth0` in the key is the supported convention for 0.2.0. |

Do **not** also set the same static address in CloudInit `networkData` — that races with offline rewrite. CloudInit `userData` (hostname, scripts) is fine.

---

## 6. First rewrite

1. Leave the VM stopped, or use the first start after clone.
2. Confirm `enabled=true` and the `*-ip` (and optional `dns`) annotations on the template.
3. Start the VM (`virtctl start` or `runStrategy: Always`).
4. The webhook injects init container `ip-rewrite` → libguestfs rewrites the disk → the guest boots.

Expect extra time before the guest can start: roughly **40–50 seconds** on RHEL and **17–18 seconds** on Windows for the offline rewrite. Later restarts with the same IP skip that cost (see below).

---

## 7. Verify

**Injection (operator logs):**

```bash
oc -n ip-rewrite-operator logs deploy/ip-rewrite-operator --tail=100 \
  | grep -E 'Injected IP rewrite|skipping'
```

Look for `Injected IP rewrite init container` and image tag `init-0.2.0`.

**Init container:**

```bash
POD=$(oc get pod -n <vm-namespace> -l kubevirt.io/domain=<vm-name> \
  -o jsonpath='{.items[0].metadata.name}')
oc logs -n <vm-namespace> "$POD" -c ip-rewrite
```

Expect successful OS detection and `IP rewrite entrypoint completed successfully`.

**Guest:** the NIC has the requested IPv4. With the guest agent connected, the operator can stamp `applied` automatically.

---

## 8. Unchanged IP (no-op restart)

When the guest agent reports the desired IPv4 on a non-loopback NIC, the operator writes `iprewrite.reip.io/applied` on the VM template (a fingerprint of the desired `*-ip` / `dns` annotations). The next stop/start with the **same** IP and DNS **skips** init injection.

Expect operator logs mentioning an applied fingerprint match, and **no** `ip-rewrite` init on the virt-launcher pod.

---

## 9. Change the IP again

Edit `iprewrite.reip.io/<iface>-ip` and/or `dns` on the template. The fingerprint no longer matches, so the next start injects and rewrites again. After the guest comes up on the new address, `applied` is updated automatically.

---

## 10. Limitations

- IPv4 only; no hostname rewrite; no BitLocker
- PVC-backed disks only; live migration pods are skipped
- Webhook failure policy is `Ignore` (fail-open if the webhook is unavailable)
- Guests: RHEL and Windows only (not CentOS Stream)
- **0.2.0 — namespace scope:** `spec.targetNamespaces` controls **`applied` stamping**, not init injection. Any namespace where a VM has `enabled=true` can receive the init container. Do not set that label outside the namespaces you intend to manage.
- **0.2.0 — NIC identity:** selection uses the guest-oriented `<iface>` in the annotation key (exact match on Linux; logical name on single-NIC Windows). Prefer confirming the Linux device name on each image.

---

## 11. Troubleshooting

| Symptom | What to check |
|---------|----------------|
| No init injected | Label `iprewrite.reip.io/enabled=true` on **template** labels; webhook CA bundle present; `applied` does not already match the current IP/DNS |
| Init runs in an unexpected namespace | Expected in 0.2.0 whenever `enabled=true`. Remove the label outside intended namespaces |
| Linux rewrite fails / interface not found | `<iface>` must match the guest NIC (`enp1s0` vs `eth0` on RHEL 10 is a common mistake) |
| Init fails OS detection | Supported guest OS; root PVC mounted into the init; image tag `init-0.2.0` |
| `applied` never stamped | Guest agent connected; expected IPv4 visible to the agent; VM namespace listed in `targetNamespaces`; label still `enabled=true` |
| IP does not stick after boot | CloudInit `networkData` (or similar) overriding the offline config — remove conflicting guest network cloud-init |
| Windows shows Install Windows / “unexpected restart” | Disk left mid-Setup from a short verify boot. Re-clone from a finished golden image or snapshot |
| Image pull errors from Quay | Node can reach `quay.io`; if using a private mirror/repo, pull secret + `imagePullSecrets` on the operator SA/Deployment |
| Odd webhook or duplicate operator behavior | Do not run a kustomize Deployment and an OLM Subscription for this operator at the same time |

---

## Further reading

| Document | Audience |
|----------|----------|
| [architecture.md](architecture.md) | How webhook, init, and applied-stamping fit together |
| [olm.md](olm.md) | OLM packaging and catalog details (maintainers / advanced install) |
| [hand-install-0.2.0.md](hand-install-0.2.0.md) | Step-by-step OLM lab hand-install notes |
| [test-records/](test-records/README.md) | Scrubbed beta validation matrix and gate |
| [ROADMAP.md](ROADMAP.md) | Planned follow-ups after 0.2.0 |
