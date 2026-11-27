# Beta 0.2.0 — matrix results and restart overhead

Living record for Phase 5b validation. Update after each OS suite.  
**No lab secrets, cluster URLs, or credentials belong here.**

Overhead is intentional: opted-in VMs pay a measurable libguestfs init cost on rewrite; noop restarts (matching `applied` fingerprint) skip injection.

**Operator:** 0.2.0 · **Init tag:** `init-0.2.0`  
**Last updated:** 2026-11-27 · **Gate:** [beta-0.2.0-gate.md](beta-0.2.0-gate.md)

## Results + restart overhead

| OS | first-start | noop | change-ip | 2nd noop | reIP init (guestfs) | noop → Running | Δ (reIP − noop) |
|---|---|---|---|---|---|---|---|
| RHEL 8 | PASS | PASS | PASS | PASS (~23s) | ~50s | ~0s init | **~50s** |
| RHEL 9 | PASS | PASS | PASS | PASS (~23s) | **~47s** | ~0s init | **~47s** |
| RHEL 10 | PASS | PASS | PASS | PASS (~23s) | ~40s† | ~0s init | **~40s** |
| Win 2016 | PASS | PASS | PASS | PASS (~23s) | **~17s** | ~0s init | **~17s** |
| Win 2019 | PASS | PASS | PASS | PASS (~23s) | **~18s** | ~0s init | **~18s** |
| Win 2022 | PASS | PASS | PASS | PASS (~23s) | **~17s** | ~0s init | **~17s** |
| Win 2025 | PASS | PASS | PASS | PASS (~23s) | **~18s** | ~0s init | **~18s** |

† Approximate from start→init `exit=0` polling when no init log was kept. Prefer log wall-clock.

## Mass parallel (7 VMs)

| Path | Wall clock (all done) | Per-VM finish (range) | Result |
|---|---|---|---|
| Parallel **rewrite** (new IPs + clear `applied`) | **120s** | ~75–120s | **PASS** all 7 |
| Parallel **noop** (unchanged) | **67s** | ~15–67s | **PASS** all 7 (no reinject) |

Impact: parallel rewrite wall (~120s) is much closer to the slowest single guestfs+boot than 7× sequential (~7×47s RHEL-class). Noop parallel wall (~67s) has **no** guestfs tax; dominated by virt-launcher start.

## Scope control

| Check | Result | Detail |
|---|---|---|
| Granular: target NS, IP ann, **no** `enabled` label | **PASS** | No `ip-rewrite` init, no `applied` |
| Namespace: NS **not** in `targetNamespaces`, but `enabled=true` | **FAIL** (known limitation) | Init **was** injected. Webhook only matches `enabled` label; **`targetNamespaces` is enforced for `applied` stamping, not for init injection.** Tracked as a **bug / fix in 0.2.1** (webhook must skip mutate when pod NS ∉ `IPRewriteConfig.spec.targetNamespaces`). Acceptable for 0.2.0 beta only if operators treat `enabled` as cluster-wide opt-in and keep labels off outside intended NS. |

## How timings are measured

| Metric | Definition |
|---|---|
| **reIP init (guestfs)** | Wall clock from init log `entrypoint starting` → `entrypoint completed successfully` (preferred). Fallback: time until init `exit=0`. |
| **noop → Running** | Restart with matching `applied`: **no** `ip-rewrite` init. Guestfs cost = **0s**. |
| **Δ (reIP − noop)** | Extra delay from offline rewrite before the guest can boot. Document, do not hide. |

## Opt-in

Webhook matches pod **label** `iprewrite.reip.io/enabled=true` on `spec.template.metadata.labels`. Annotation alone does not inject.

## Lab pitfalls (not product FAIL)

Short boot-verify halt on Windows clones still in Setup/specialize leaves a dirty PVC; next boot shows Install Windows “unexpected restart”. Fresh clone from the golden/snapshot boots to OOBE. Do not treat that as ODF clone corruption when a clean PVC works.
