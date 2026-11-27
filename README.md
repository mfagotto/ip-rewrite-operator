# ip-rewrite-operator

Rewrite static guest IPv4 (and optional DNS) on OpenShift Virtualization VMs **before the guest boots**. A mutating webhook injects a libguestfs init container that edits the root disk offline. This is **not** CloudInit-based reIP.

**Version:** 0.2.0 (pre-production beta) · OpenShift **4.22** / OpenShift Virtualization  
**Images:** `quay.io/mfagotto/ip-rewrite-operator:0.2.0` (manager, UBI 10) · `:init-0.2.0` (libguestfs init, CentOS Stream 10)

Designed around **OpenShift Service CA** for webhook TLS (no cert-manager required), optional SCC management, and a fail-open webhook.

Architecture inspired by [Soteria IP Rewrite](https://raffaelespazzoli.github.io/soteria/architecture/ip-rewrite/) — see [NOTICE](NOTICE).

## Documentation

| Document | Contents |
|----------|----------|
| [User guide](docs/user-guide.md) | Install, configure, opt in VMs, verify, troubleshoot, limitations |
| [Architecture](docs/architecture.md) | Webhook, init, and applied-stamping flow |
| [Examples](examples/vm-snippet.yaml) | Lean Linux and Windows VM annotation fragments |
| [OLM](docs/olm.md) | Bundle / catalog packaging and OLM install notes |
| [Test records](docs/test-records/README.md) | Scrubbed beta validation matrix and gate |
| [Roadmap](docs/ROADMAP.md) | Planned work after 0.2.0 |

## Repository

- Origin: `https://github.com/mfagotto/ip-rewrite-operator.git`
- Layout: operator (`cmd/`, `internal/`), init image (`images/init/`), manifests (`config/`), OLM (`bundle/`), docs (`docs/`)

## Develop

```bash
make test    # unit tests
make build   # compile the manager
make fmt vet
```

Container images (when ready to push):

```bash
make docker-build        # manager
make docker-build-init   # init
```

OLM bundle checks: `make bundle-validate` (see [docs/olm.md](docs/olm.md)).

**Install and day-2 usage** (kustomize, OLM, VM annotations, limitations): follow the [user guide](docs/user-guide.md) — do not duplicate that path here.

## Validation

- Unit: `make test`
- Cluster / VM e2e evidence: [docs/test-records/README.md](docs/test-records/README.md)

## Status and next steps

**0.2.0** is the current beta line. Known limits and adoption guidance live in the [user guide](docs/user-guide.md). Planned follow-ups (namespace-scope fix, NIC identity by MAC, and more): [roadmap](docs/ROADMAP.md).
