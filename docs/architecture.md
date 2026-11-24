# Architecture

Offline guest IP rewrite for OpenShift Virtualization:

1. User sets `iprewrite.windowsreip.io/*` labels/annotations on the VM pod template.
2. On stop/start, KubeVirt creates a virt-launcher pod carrying those labels.
3. The MutatingWebhookConfiguration (objectSelector on `enabled=true`) sends Pod CREATE to the operator webhook.
4. Webhook skips live-migration pods; otherwise injects an `ip-rewrite` init container with PVC mounts/devices and `IPREWRITE_*` env.
5. Init container (libguestfs direct backend) rewrites RHEL (Augeas) or Windows (hivex) config on disk, then QEMU starts.

Webhook TLS uses OpenShift Service CA (`serving-cert-secret-name` + `inject-cabundle`). Failure policy is `Ignore` (fail-open).
