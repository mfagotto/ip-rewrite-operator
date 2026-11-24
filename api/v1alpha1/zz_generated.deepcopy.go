/*
Copyright 2026 The ip-rewrite-operator Authors.
Licensed under the Apache License, Version 2.0.
*/

// Code generated manually for bootstrap; regenerate with controller-gen when available.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto copies the receiver into out.
func (in *IPRewriteConfig) DeepCopyInto(out *IPRewriteConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy creates a new deep copy.
func (in *IPRewriteConfig) DeepCopy() *IPRewriteConfig {
	if in == nil {
		return nil
	}
	out := new(IPRewriteConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements runtime.Object.
func (in *IPRewriteConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies the receiver into out.
func (in *IPRewriteConfigList) DeepCopyInto(out *IPRewriteConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IPRewriteConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a new deep copy.
func (in *IPRewriteConfigList) DeepCopy() *IPRewriteConfigList {
	if in == nil {
		return nil
	}
	out := new(IPRewriteConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements runtime.Object.
func (in *IPRewriteConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies the receiver into out.
func (in *IPRewriteConfigSpec) DeepCopyInto(out *IPRewriteConfigSpec) {
	*out = *in
	if in.TargetNamespaces != nil {
		in, out := &in.TargetNamespaces, &out.TargetNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.VirtLauncherServiceAccountNames != nil {
		in, out := &in.VirtLauncherServiceAccountNames, &out.VirtLauncherServiceAccountNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.RequestKVMDevice != nil {
		in, out := &in.RequestKVMDevice, &out.RequestKVMDevice
		*out = new(bool)
		**out = **in
	}
	if in.ManageSCC != nil {
		in, out := &in.ManageSCC, &out.ManageSCC
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy creates a new deep copy.
func (in *IPRewriteConfigSpec) DeepCopy() *IPRewriteConfigSpec {
	if in == nil {
		return nil
	}
	out := new(IPRewriteConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver into out.
func (in *IPRewriteConfigStatus) DeepCopyInto(out *IPRewriteConfigStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a new deep copy.
func (in *IPRewriteConfigStatus) DeepCopy() *IPRewriteConfigStatus {
	if in == nil {
		return nil
	}
	out := new(IPRewriteConfigStatus)
	in.DeepCopyInto(out)
	return out
}
