/*
Copyright 2026 The ip-rewrite-operator Authors.
Licensed under the Apache License, Version 2.0.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IPRewriteConfigSpec defines the desired state of IPRewriteConfig.
type IPRewriteConfigSpec struct {
	// TargetNamespaces are namespaces where virt-launcher pods may run with
	// IP rewrite enabled. The operator grants SCC use to virt-launcher
	// ServiceAccounts in these namespaces when ManageSCC is true.
	// +optional
	TargetNamespaces []string `json:"targetNamespaces,omitempty"`

	// VirtLauncherServiceAccountNames are ServiceAccount names used by
	// virt-launcher pods (default: ["default"]).
	// +optional
	VirtLauncherServiceAccountNames []string `json:"virtLauncherServiceAccountNames,omitempty"`

	// InitContainerImage is the image injected into virt-launcher pods.
	// +optional
	InitContainerImage string `json:"initContainerImage,omitempty"`

	// InitContainerPullPolicy is the imagePullPolicy for the init container.
	// +kubebuilder:default=IfNotPresent
	// +optional
	InitContainerPullPolicy string `json:"initContainerPullPolicy,omitempty"`

	// RequestKVMDevice requests devices.kubevirt.io/kvm for the init container.
	// +kubebuilder:default=true
	// +optional
	RequestKVMDevice *bool `json:"requestKVMDevice,omitempty"`

	// ManageSCC controls whether the operator creates/updates an SCC for
	// virt-launcher ServiceAccounts in targetNamespaces.
	// +kubebuilder:default=true
	// +optional
	ManageSCC *bool `json:"manageSCC,omitempty"`

	// WebhookFailurePolicy is informational; the MWC is installed as Ignore (fail-open).
	// +kubebuilder:default=Ignore
	// +kubebuilder:validation:Enum=Ignore;Fail
	// +optional
	WebhookFailurePolicy string `json:"webhookFailurePolicy,omitempty"`
}

// IPRewriteConfigStatus defines the observed state of IPRewriteConfig.
type IPRewriteConfigStatus struct {
	// Conditions represent the latest available observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=irc
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// IPRewriteConfig is the Schema for the iprewriteconfigs API (cluster singleton).
type IPRewriteConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPRewriteConfigSpec   `json:"spec,omitempty"`
	Status IPRewriteConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IPRewriteConfigList contains a list of IPRewriteConfig.
type IPRewriteConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPRewriteConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IPRewriteConfig{}, &IPRewriteConfigList{})
}
