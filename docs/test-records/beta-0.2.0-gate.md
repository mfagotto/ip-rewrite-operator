# Beta 0.2.0 — quality closeout / gate

**Status:** **READY for pre-production beta** (with known limitation below).  
**Date:** 2026-11-27 · **Operator / init:** `0.2.0` / `init-0.2.0`

## Evidence

Living results table (per-OS + mass + scope): [beta-0.2.0-matrix.md](beta-0.2.0-matrix.md).

| Gate item | Result |
|-----------|--------|
| 7-OS first-start / noop / change-IP / 2nd noop | PASS |
| Mass parallel rewrite (7 VMs) | PASS (~120s wall) |
| Mass parallel noop (7 VMs) | PASS (~67s wall) |
| Scope: no `enabled` label | PASS |
| Scope: NS outside `targetNamespaces` | FAIL → **known limit** → fix **0.2.1** |
| ODF RWX testbeds / Win2022 golden exercised | PASS |

## Known limitation (accepted for 0.2.0)

Webhook init injection does **not** enforce `IPRewriteConfig.spec.targetNamespaces` (label `enabled=true` is enough). Mitigation: never set that label outside intended namespaces. Tracked in 0.2.1 Phase 0a ([roadmap](../ROADMAP.md)).

## Product docs synced

- README limitations
- User guide §10–12
- This gate + matrix under `docs/test-records/`

## Beta declaration

**ip-rewrite-operator 0.2.0** is acceptable for pre-prod OpenShift Virtualization use under the documented API and the namespace-scope caveat above. OperatorHub packaging and webhook NS enforcement are **out of scope** for this beta (see 0.2.1 plan).
