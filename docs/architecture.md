# Architecture

Offline guest IP rewrite for OpenShift Virtualization. The operator intercepts virt-launcher pod creation, injects a libguestfs init container that edits the root disk **before** QEMU starts, then stamps an `applied` fingerprint when the guest agent confirms the new IPv4 — so unchanged restarts skip rewrite.

API and day-2 usage: [user guide](user-guide.md). Examples: [vm-snippet.yaml](../examples/vm-snippet.yaml).

Design inspired by [Soteria IP Rewrite](https://raffaelespazzoli.github.io/soteria/architecture/ip-rewrite/) — see [NOTICE](../NOTICE).

---

## End-to-end flow

```mermaid
sequenceDiagram
  autonumber
  participant User as User / GitOps
  participant VM as VirtualMachine
  participant API as kube-apiserver
  participant WH as IP Rewrite webhook
  participant Pod as virt-launcher Pod
  participant Init as ip-rewrite init
  participant Disk as Guest disk<br/>(libguestfs)
  participant Guest as Guest OS + agent
  participant Ctrl as VMApplied controller

  User->>VM: Label enabled=true<br/>Annotate iprewrite.reip.io/*-ip (+ dns)
  User->>VM: Stop / start (or first boot)

  Note over VM,API: KubeVirt creates virt-launcher<br/>(inherits template labels + annotations)

  API->>API: MWC objectSelector<br/>enabled=true
  API->>WH: AdmissionReview (Pod CREATE)

  WH->>WH: Skip if live migration
  WH->>WH: Skip if applied == desired fingerprint
  WH->>WH: Map annotations → IPREWRITE_* env<br/>Mount/attach PVC disks

  alt Rewrite needed
    WH-->>API: JSON patch (inject ip-rewrite init)
    API->>Pod: Create mutated pod
    Note over Pod,Init: Init runs before QEMU
    Init->>Disk: virt-inspector (OS detect)
    alt RHEL / Linux
      Init->>Disk: Augeas / NM / ifcfg rewrite
    else Windows
      Init->>Disk: hivex SYSTEM hive rewrite
    end
    Init-->>Pod: Exit 0
    Pod->>Guest: QEMU starts → guest boots with new IP
    Guest-->>Ctrl: Guest agent reports IPv4
    Ctrl->>VM: Stamp iprewrite.reip.io/applied
  else No-op (fingerprint match)
    WH-->>API: Allow unmodified
    API->>Pod: Create pod (no ip-rewrite init)
    Pod->>Guest: QEMU starts immediately
  end
```

### Later restart with the same IP

```mermaid
flowchart LR
  A[Stop / start VM] --> B{applied equals<br/>desired fingerprint?}
  B -->|yes| C[Webhook skips inject]
  C --> D[Guest boots — no guestfs tax]
  B -->|no<br/>IP or DNS changed| E[Inject ip-rewrite init]
  E --> F[Offline rewrite]
  F --> G[Boot + stamp new applied]
```

---

## Components

| Piece | Role |
|-------|------|
| Mutating webhook (`internal/webhook`) | On Pod CREATE with `enabled=true`: skip migration / matching `applied`; otherwise inject `ip-rewrite` init with disk mounts and env |
| Init image (`images/init`) | libguestfs direct backend: OS detect → RHEL (Augeas) or Windows (`virt-windows-ip-rewrite` / hivex) |
| VMApplied controller (`internal/controller`) | When guest agent shows expected IPv4, patch template `applied` fingerprint |
| `IPRewriteConfig` | Cluster config: init image, SCC, `targetNamespaces` (gates **applied** stamping in 0.2.0; see user guide) |

---

## Notes

1. Labels/annotations must sit on **`spec.template.metadata`** so KubeVirt copies them onto the virt-launcher pod.
2. Webhook failure policy is **`Ignore`** (fail-open): if the webhook is down, pods admit without rewrite.
3. Webhook TLS: OLM-injected cert at `/tmp/k8s-webhook-server/serving-certs`, or OpenShift Service CA on kustomize installs.
4. OLM packaging uses CSV `webhookdefinitions` + webhook Service; SCC and sample config are applied outside the registry+v1 bundle — see [olm.md](olm.md).
