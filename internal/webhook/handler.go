/*
Copyright 2026 The ip-rewrite-operator Authors.
Licensed under the Apache License, Version 2.0.

Design inspired by the Soteria IP rewrite webhook (Apache-2.0);
see NOTICE for attribution.
*/

// Package webhook injects an IP rewrite init container into virt-launcher pods.
package webhook

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	virtv1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	// EnabledLabel opts a VM (via pod template) into IP rewrite.
	EnabledLabel = "iprewrite.windowsreip.io/enabled"

	annotationPrefix = "iprewrite.windowsreip.io/"
	dnsSuffix        = "dns"
	ipSuffix         = "-ip"

	initContainerName = "ip-rewrite"
	diskMountPrefix   = "/disks/"
	diskDevicePrefix  = "/disks/"
	kvmDeviceResource = "devices.kubevirt.io/kvm"

	// DefaultInitContainerImage used when not configured.
	DefaultInitContainerImage = "ghcr.io/mfagotto/ip-rewrite-init:latest"

	// MutatePodPath is the admission webhook HTTP path.
	MutatePodPath = "/mutate-v1-pod"
)

// Handler mutates pods labeled for IP rewrite.
type Handler struct {
	InitContainerImage      string
	RequestKVMDevice        bool
	InitContainerPullPolicy corev1.PullPolicy
}

// Handle processes Pod CREATE admission requests (fail-open on internal errors).
func (h *Handler) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx)

	pod := &corev1.Pod{}
	if err := json.Unmarshal(req.Object.Raw, pod); err != nil {
		logger.Error(err, "Failed to unmarshal Pod, admitting unmodified (fail-open)")
		return admission.Allowed("")
	}

	logger = logger.WithValues("pod", pod.Name, "namespace", req.Namespace)

	if _, isMigration := pod.Labels[virtv1.MigrationJobLabel]; isMigration {
		logger.Info("Skipping injection for migration pod")
		return admission.Allowed("")
	}

	for _, c := range pod.Spec.InitContainers {
		if c.Name == initContainerName {
			logger.Info("Init container already present, skipping injection")
			return admission.Allowed("")
		}
	}

	envVars := annotationsToEnvVars(pod.Annotations)
	if len(envVars) == 0 {
		logger.Info("No IP rewrite annotations found, skipping injection")
		return admission.Allowed("")
	}

	image := h.InitContainerImage
	if image == "" {
		image = DefaultInitContainerImage
	}
	pullPolicy := h.InitContainerPullPolicy
	if pullPolicy == "" {
		pullPolicy = corev1.PullIfNotPresent
	}

	blockVolumes := blockModeVolumes(pod)
	volumeMounts, volumeDevices := pvcVolumeAccess(pod.Spec.Volumes, blockVolumes)

	guestfsEnv := []corev1.EnvVar{
		{Name: "LIBGUESTFS_BACKEND", Value: "direct"},
		{Name: "LIBGUESTFS_PATH", Value: "/guestfs-appliance"},
	}
	envVars = append(guestfsEnv, envVars...)

	runAsUser := int64(107)
	runAsNonRoot := true
	allowPrivEsc := false
	initContainer := corev1.Container{
		Name:            initContainerName,
		Image:           image,
		ImagePullPolicy: pullPolicy,
		Env:             envVars,
		VolumeMounts:    volumeMounts,
		VolumeDevices:   volumeDevices,
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:                &runAsUser,
			RunAsNonRoot:             &runAsNonRoot,
			AllowPrivilegeEscalation: &allowPrivEsc,
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
	}

	if h.RequestKVMDevice {
		initContainer.Resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceName(kvmDeviceResource): resource.MustParse("1"),
			},
		}
	}

	pod.Spec.InitContainers = append([]corev1.Container{initContainer}, pod.Spec.InitContainers...)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		logger.Error(err, "Failed to marshal mutated Pod, admitting unmodified (fail-open)")
		return admission.Allowed("")
	}

	logger.Info("Injected IP rewrite init container",
		"image", image,
		"envVars", len(envVars),
		"volumeMounts", len(volumeMounts),
		"volumeDevices", len(volumeDevices))

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func annotationsToEnvVars(annotations map[string]string) []corev1.EnvVar {
	keys := make([]string, 0, len(annotations))
	for key := range annotations {
		if strings.HasPrefix(key, annotationPrefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	var envVars []corev1.EnvVar
	for _, key := range keys {
		value := annotations[key]
		suffix := key[len(annotationPrefix):]
		switch {
		case suffix == dnsSuffix:
			envVars = append(envVars, corev1.EnvVar{Name: "IPREWRITE_DNS", Value: value})
		case strings.HasSuffix(suffix, ipSuffix):
			name := suffix[:len(suffix)-len(ipSuffix)]
			envName := "IPREWRITE_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_IP"
			envVars = append(envVars, corev1.EnvVar{Name: envName, Value: value})
		}
	}
	return envVars
}

func blockModeVolumes(pod *corev1.Pod) map[string]bool {
	blocks := make(map[string]bool)
	for _, c := range pod.Spec.Containers {
		for _, vd := range c.VolumeDevices {
			blocks[vd.Name] = true
		}
	}
	for _, c := range pod.Spec.InitContainers {
		for _, vd := range c.VolumeDevices {
			blocks[vd.Name] = true
		}
	}
	return blocks
}

func pvcVolumeAccess(volumes []corev1.Volume, blockVolumes map[string]bool) ([]corev1.VolumeMount, []corev1.VolumeDevice) {
	var mounts []corev1.VolumeMount
	var devices []corev1.VolumeDevice
	for _, vol := range volumes {
		if vol.PersistentVolumeClaim == nil {
			continue
		}
		if blockVolumes[vol.Name] {
			devices = append(devices, corev1.VolumeDevice{
				Name:       vol.Name,
				DevicePath: diskDevicePrefix + vol.Name,
			})
		} else {
			mounts = append(mounts, corev1.VolumeMount{
				Name:      vol.Name,
				MountPath: diskMountPrefix + vol.Name,
			})
		}
	}
	return mounts, devices
}
