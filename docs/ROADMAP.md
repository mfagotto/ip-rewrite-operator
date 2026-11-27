# Roadmap

## 0.2.0 beta

Gate and matrix: [test-records/beta-0.2.0-gate.md](test-records/beta-0.2.0-gate.md) · [test-records/beta-0.2.0-matrix.md](test-records/beta-0.2.0-matrix.md).

## 0.2.1 (planned)

1. **Webhook honors `targetNamespaces`** (fix 0.2.0 scope FAIL).
2. Declarative DHCP mode (static ↔ DHCP via annotations).
3. **NIC targeting by MAC / KubeVirt `interfaces[].name`** (replace guest OS names as the primary key; fix Linux `enp*` / Windows `eth0` confusion).
4. SCC fully reconciled by the operator; Service CA webhook default; Quay bundle/catalog segregation.
5. Regression matrix including NS scope; then OperatorHub track.
6. Ascertain: `enabled=true` **without** `*-ip` annotations → no init, guest network unchanged (label-only is not a rewrite).
