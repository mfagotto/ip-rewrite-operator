/*
Copyright 2026 The ip-rewrite-operator Authors.
Licensed under the Apache License, Version 2.0.
*/

package controller

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	virtv1 "kubevirt.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	iprewritev1alpha1 "github.com/mfagotto/ip-rewrite-operator/api/v1alpha1"
	ipwebhook "github.com/mfagotto/ip-rewrite-operator/internal/webhook"
)

const (
	appliedRequeue = 15 * time.Second
	ipConfigName   = "cluster"
)

// VMAppliedReconciler stamps iprewrite.reip.io/applied on the VM template after
// the guest agent reports the desired IPv4 on a non-loopback interface.
type VMAppliedReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kubevirt.io,resources=virtualmachines,verbs=get;list;watch;patch;update
// +kubebuilder:rbac:groups=kubevirt.io,resources=virtualmachineinstances,verbs=get;list;watch
// +kubebuilder:rbac:groups=iprewrite.reip.io,resources=iprewriteconfigs,verbs=get;list;watch

func (r *VMAppliedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("vmi", req.NamespacedName)

	if !r.namespaceTargeted(ctx, req.Namespace) {
		return ctrl.Result{}, nil
	}

	vmi := &virtv1.VirtualMachineInstance{}
	if err := r.Get(ctx, req.NamespacedName, vmi); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	labels := vmi.Labels
	anns := vmi.Annotations
	if labels[ipwebhook.EnabledLabel] != "true" {
		return ctrl.Result{}, nil
	}
	if !hasDesiredIPAnnotations(anns) {
		return ctrl.Result{}, nil
	}

	desired := ipwebhook.DesiredConfigFingerprint(anns)
	if anns[ipwebhook.AppliedAnnotation] == desired {
		return ctrl.Result{}, nil
	}

	if vmi.Status.Phase != virtv1.Running {
		return ctrl.Result{RequeueAfter: appliedRequeue}, nil
	}
	if !guestAgentConnected(vmi) {
		logger.V(1).Info("waiting for guest agent")
		return ctrl.Result{RequeueAfter: appliedRequeue}, nil
	}

	expectedIPs := desiredIPv4s(anns)
	preferMAC := preferredMAC(vmi)
	if !GuestReportsExpectedIPv4(vmi.Status.Interfaces, expectedIPs, preferMAC) {
		logger.V(1).Info("guest IPs do not yet match desired",
			"expected", expectedIPs, "preferMAC", preferMAC)
		return ctrl.Result{RequeueAfter: appliedRequeue}, nil
	}

	vm := &virtv1.VirtualMachine{}
	if err := r.Get(ctx, req.NamespacedName, vm); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if vm.Spec.Template == nil {
		return ctrl.Result{}, fmt.Errorf("VM %s/%s has nil template", vm.Namespace, vm.Name)
	}
	if vm.Spec.Template.ObjectMeta.Annotations == nil {
		vm.Spec.Template.ObjectMeta.Annotations = map[string]string{}
	}
	if vm.Spec.Template.ObjectMeta.Annotations[ipwebhook.AppliedAnnotation] == desired {
		return ctrl.Result{}, nil
	}

	patch := client.MergeFrom(vm.DeepCopy())
	vm.Spec.Template.ObjectMeta.Annotations[ipwebhook.AppliedAnnotation] = desired
	if err := r.Patch(ctx, vm, patch); err != nil {
		return ctrl.Result{}, fmt.Errorf("patch VM applied annotation: %w", err)
	}
	logger.Info("stamped applied fingerprint on VM template", "fingerprint", desired)
	return ctrl.Result{}, nil
}

func (r *VMAppliedReconciler) namespaceTargeted(ctx context.Context, ns string) bool {
	list := &iprewritev1alpha1.IPRewriteConfigList{}
	if err := r.List(ctx, list); err != nil {
		return false
	}
	for i := range list.Items {
		for _, t := range list.Items[i].Spec.TargetNamespaces {
			if t == ns {
				return true
			}
		}
	}
	// Fallback: conventional singleton name.
	cfg := &iprewritev1alpha1.IPRewriteConfig{}
	if err := r.Get(ctx, types.NamespacedName{Name: ipConfigName}, cfg); err != nil {
		return false
	}
	for _, t := range cfg.Spec.TargetNamespaces {
		if t == ns {
			return true
		}
	}
	return false
}

// SetupWithManager registers the controller.
func (r *VMAppliedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&virtv1.VirtualMachineInstance{}).
		Watches(&virtv1.VirtualMachine{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
			return []reconcile.Request{{
				NamespacedName: types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()},
			}}
		})).
		Complete(r)
}

func hasDesiredIPAnnotations(anns map[string]string) bool {
	for k := range anns {
		if strings.HasPrefix(k, ipwebhook.AnnotationPrefix) &&
			strings.HasSuffix(k, "-ip") &&
			k != ipwebhook.AppliedAnnotation {
			return true
		}
	}
	return false
}

func desiredIPv4s(anns map[string]string) []string {
	var out []string
	for k, v := range anns {
		if !strings.HasPrefix(k, ipwebhook.AnnotationPrefix) || k == ipwebhook.AppliedAnnotation {
			continue
		}
		suf := strings.TrimPrefix(k, ipwebhook.AnnotationPrefix)
		if !strings.HasSuffix(suf, "-ip") {
			continue
		}
		if ip := parseAnnotationIPv4(v); ip != "" {
			out = append(out, ip)
		}
	}
	return out
}

// parseAnnotationIPv4 extracts the IPv4 from values like "192.168.100.60/24;192.168.100.1".
func parseAnnotationIPv4(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	host := value
	if i := strings.IndexAny(host, "/;"); i >= 0 {
		host = host[:i]
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil || ip.To4() == nil {
		return ""
	}
	return ip.To4().String()
}

func preferredMAC(vmi *virtv1.VirtualMachineInstance) string {
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.MacAddress != "" {
			return strings.ToLower(iface.MacAddress)
		}
	}
	return ""
}

func guestAgentConnected(vmi *virtv1.VirtualMachineInstance) bool {
	for _, c := range vmi.Status.Conditions {
		if c.Type == virtv1.VirtualMachineInstanceAgentConnected && c.Status == "True" {
			return true
		}
	}
	// Fallback: guest-agent info source on any interface.
	for _, iface := range vmi.Status.Interfaces {
		if strings.Contains(strings.ToLower(iface.InfoSource), "guest-agent") {
			return true
		}
	}
	return false
}

// GuestReportsExpectedIPv4 returns true when every expected IPv4 appears on a
// non-loopback guest interface. Prefers the Multus NIC MAC when provided.
func GuestReportsExpectedIPv4(ifaces []virtv1.VirtualMachineInstanceNetworkInterface, expected []string, preferMAC string) bool {
	if len(expected) == 0 {
		return false
	}
	reported := collectGuestIPv4s(ifaces, preferMAC)
	if len(reported) == 0 {
		return false
	}
	for _, want := range expected {
		found := false
		for _, got := range reported {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func collectGuestIPv4s(ifaces []virtv1.VirtualMachineInstanceNetworkInterface, preferMAC string) []string {
	preferMAC = strings.ToLower(preferMAC)
	var preferred, primary, anyNonLo []string

	for _, iface := range ifaces {
		if isLoopbackIface(iface) {
			continue
		}
		ips := ipv4sFromIface(iface)
		if len(ips) == 0 {
			continue
		}
		anyNonLo = append(anyNonLo, ips...)
		mac := strings.ToLower(iface.MAC)
		if preferMAC != "" && mac == preferMAC {
			preferred = append(preferred, ips...)
		}
		name := strings.ToLower(iface.InterfaceName)
		if name == "eth0" || name == "ethernet" || strings.HasPrefix(name, "ethernet ") {
			primary = append(primary, ips...)
		}
	}

	switch {
	case len(preferred) > 0:
		return uniqueStrings(preferred)
	case len(primary) > 0:
		return uniqueStrings(primary)
	default:
		return uniqueStrings(anyNonLo)
	}
}

func isLoopbackIface(iface virtv1.VirtualMachineInstanceNetworkInterface) bool {
	name := strings.ToLower(iface.InterfaceName)
	if name == "lo" || name == "loopback" || strings.Contains(name, "loopback") {
		return true
	}
	for _, raw := range append([]string{iface.IP}, iface.IPs...) {
		ip := net.ParseIP(stripCIDR(raw))
		if ip != nil && ip.IsLoopback() {
			return true
		}
	}
	return false
}

func ipv4sFromIface(iface virtv1.VirtualMachineInstanceNetworkInterface) []string {
	var out []string
	for _, raw := range append([]string{iface.IP}, iface.IPs...) {
		ip := net.ParseIP(stripCIDR(raw))
		if ip == nil || ip.To4() == nil || ip.IsLoopback() {
			continue
		}
		out = append(out, ip.To4().String())
	}
	return uniqueStrings(out)
}

func stripCIDR(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "/"); i >= 0 {
		return s[:i]
	}
	return s
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
