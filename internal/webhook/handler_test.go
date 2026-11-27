/*
Copyright 2026 The ip-rewrite-operator Authors.
Licensed under the Apache License, Version 2.0.
*/

package webhook

import (
	"context"
	"encoding/json"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	virtv1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestAnnotationsToEnvVars(t *testing.T) {
	env := annotationsToEnvVars(map[string]string{
		"iprewrite.reip.io/eth0-ip": "10.0.2.100/24;10.0.2.1",
		"iprewrite.reip.io/dns":    "10.0.2.10,10.0.2.11",
		"other.io/foo":                    "bar",
	})
	if len(env) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(env))
	}
	if env[0].Name != "IPREWRITE_DNS" || env[0].Value != "10.0.2.10,10.0.2.11" {
		t.Fatalf("unexpected DNS env: %+v", env[0])
	}
	if env[1].Name != "IPREWRITE_ETH0_IP" || env[1].Value != "10.0.2.100/24;10.0.2.1" {
		t.Fatalf("unexpected eth0 env: %+v", env[1])
	}
}

func TestHandleMigrationSkip(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "virt-launcher-x",
			Labels: map[string]string{
				EnabledLabel:             "true",
				virtv1.MigrationJobLabel: "abc",
			},
			Annotations: map[string]string{
				"iprewrite.reip.io/eth0-ip": "10.0.2.100/24;10.0.2.1",
			},
		},
	}
	raw, _ := json.Marshal(pod)
	h := &Handler{InitContainerImage: "test:latest", RequestKVMDevice: true}
	resp := h.Handle(context.Background(), admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{Raw: raw},
		},
	})
	if !resp.Allowed || len(resp.Patches) != 0 {
		t.Fatalf("expected allow with no patches for migration pod, got allowed=%v patches=%d", resp.Allowed, len(resp.Patches))
	}
}

func TestDesiredConfigFingerprintStable(t *testing.T) {
	a := map[string]string{
		"iprewrite.reip.io/dns":     "8.8.8.8,1.1.1.1",
		"iprewrite.reip.io/eth0-ip": "192.168.100.60/24;192.168.100.1",
		AppliedAnnotation:                  "should-be-ignored",
		"unrelated":                        "x",
	}
	b := map[string]string{
		"iprewrite.reip.io/eth0-ip": "192.168.100.60/24;192.168.100.1",
		"iprewrite.reip.io/dns":     "8.8.8.8,1.1.1.1",
	}
	fa, fb := DesiredConfigFingerprint(a), DesiredConfigFingerprint(b)
	if fa == "" || fa != fb {
		t.Fatalf("expected stable fingerprint, got %q vs %q", fa, fb)
	}
	b["iprewrite.reip.io/eth0-ip"] = "192.168.100.61/24;192.168.100.1"
	if DesiredConfigFingerprint(b) == fa {
		t.Fatal("fingerprint should change when IP changes")
	}
}

func TestHandleSkipsWhenAppliedMatches(t *testing.T) {
	anns := map[string]string{
		"iprewrite.reip.io/eth0-ip": "10.0.2.100/24;10.0.2.1",
		"iprewrite.reip.io/dns":     "10.0.2.10,10.0.2.11",
	}
	anns[AppliedAnnotation] = DesiredConfigFingerprint(anns)
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "virt-launcher-x",
			Labels:      map[string]string{EnabledLabel: "true"},
			Annotations: anns,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{{
				Name: "rootdisk",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "rootdisk"},
				},
			}},
			Containers: []corev1.Container{{Name: "compute", Image: "kubevirt/virt-launcher"}},
		},
	}
	raw, _ := json.Marshal(pod)
	h := &Handler{InitContainerImage: "init:dev"}
	resp := h.Handle(context.Background(), admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{Raw: raw},
		},
	})
	if !resp.Allowed || len(resp.Patches) != 0 {
		t.Fatalf("expected allow with no patches when applied matches, got allowed=%v patches=%d", resp.Allowed, len(resp.Patches))
	}
}

func TestHandleInjectsInit(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "virt-launcher-x",
			Labels: map[string]string{
				EnabledLabel: "true",
			},
			Annotations: map[string]string{
				"iprewrite.reip.io/eth0-ip": "10.0.2.100/24;10.0.2.1",
			},
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{{
				Name: "rootdisk",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "rootdisk"},
				},
			}},
			Containers: []corev1.Container{{Name: "compute", Image: "kubevirt/virt-launcher"}},
		},
	}
	raw, _ := json.Marshal(pod)
	h := &Handler{
		InitContainerImage:      "init:dev",
		InitContainerPullPolicy: corev1.PullIfNotPresent,
		RequestKVMDevice:        true,
	}
	resp := h.Handle(context.Background(), admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{Raw: raw},
		},
	})
	if !resp.Allowed {
		t.Fatalf("expected allowed")
	}
	if len(resp.Patches) == 0 {
		t.Fatalf("expected JSON patches for init injection")
	}
}

func TestHandleBlockVolumeDevice(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"iprewrite.reip.io/eth0-ip": "10.0.2.100/24;10.0.2.1",
			},
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{{
				Name: "rootdisk",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "rootdisk"},
				},
			}},
			Containers: []corev1.Container{{
				Name: "compute",
				VolumeDevices: []corev1.VolumeDevice{{
					Name:       "rootdisk",
					DevicePath: "/dev/rootdisk",
				}},
			}},
		},
	}
	blocks := blockModeVolumes(&pod)
	if !blocks["rootdisk"] {
		t.Fatal("expected rootdisk as block")
	}
	mounts, devices := pvcVolumeAccess(pod.Spec.Volumes, blocks)
	if len(mounts) != 0 || len(devices) != 1 || devices[0].DevicePath != "/disks/rootdisk" {
		t.Fatalf("mounts=%v devices=%v", mounts, devices)
	}
}
