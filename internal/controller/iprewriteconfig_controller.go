/*
Copyright 2026 The ip-rewrite-operator Authors.
Licensed under the Apache License, Version 2.0.
*/

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	iprewritev1alpha1 "github.com/mfagotto/ip-rewrite-operator/api/v1alpha1"
)

const (
	sccName            = "ip-rewrite-operator"
	sccClusterRoleName = "ip-rewrite-operator-scc"
)

// IPRewriteConfigReconciler reconciles IPRewriteConfig (SCC bindings for target namespaces).
type IPRewriteConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=iprewrite.reip.io,resources=iprewriteconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=iprewrite.reip.io,resources=iprewriteconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch

func (r *IPRewriteConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	cfg := &iprewritev1alpha1.IPRewriteConfig{}
	if err := r.Get(ctx, req.NamespacedName, cfg); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	manageSCC := true
	if cfg.Spec.ManageSCC != nil {
		manageSCC = *cfg.Spec.ManageSCC
	}
	if !manageSCC {
		logger.Info("ManageSCC disabled; skipping SCC reconcile")
		return ctrl.Result{}, nil
	}

	saNames := cfg.Spec.VirtLauncherServiceAccountNames
	if len(saNames) == 0 {
		saNames = []string{"default"}
	}
	namespaces := cfg.Spec.TargetNamespaces
	if len(namespaces) == 0 {
		logger.Info("No targetNamespaces configured")
		return ctrl.Result{}, nil
	}

	subjects := make([]rbacv1.Subject, 0, len(namespaces)*len(saNames))
	for _, ns := range namespaces {
		for _, sa := range saNames {
			subjects = append(subjects, rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: ns,
			})
		}
	}

	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: sccClusterRoleName},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cr, func() error {
		cr.Rules = []rbacv1.PolicyRule{{
			APIGroups:     []string{"security.openshift.io"},
			Resources:     []string{"securitycontextconstraints"},
			ResourceNames: []string{sccName},
			Verbs:         []string{"use"},
		}}
		return controllerutil.SetControllerReference(cfg, cr, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile ClusterRole: %w", err)
	}

	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: sccClusterRoleName},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, crb, func() error {
		crb.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     sccClusterRoleName,
		}
		crb.Subjects = subjects
		return controllerutil.SetControllerReference(cfg, crb, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("reconcile ClusterRoleBinding: %w", err)
	}

	// Ensure namespaces exist (soft check).
	for _, ns := range namespaces {
		n := &corev1.Namespace{}
		if err := r.Get(ctx, types.NamespacedName{Name: ns}, n); err != nil {
			if apierrors.IsNotFound(err) {
				logger.Info("target namespace missing", "namespace", ns)
				continue
			}
			return ctrl.Result{}, err
		}
	}

	logger.Info("Reconciled SCC bindings", "namespaces", namespaces, "serviceAccounts", saNames)
	return ctrl.Result{}, nil
}

// SetupWithManager registers the controller.
func (r *IPRewriteConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iprewritev1alpha1.IPRewriteConfig{}).
		Complete(r)
}
