/*
Copyright 2026 The ip-rewrite-operator Authors.
Licensed under the Apache License, Version 2.0.
*/

package main

import (
	"crypto/tls"
	"flag"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	virtv1 "kubevirt.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	iprewritev1alpha1 "github.com/mfagotto/ip-rewrite-operator/api/v1alpha1"
	"github.com/mfagotto/ip-rewrite-operator/internal/controller"
	ipwebhook "github.com/mfagotto/ip-rewrite-operator/internal/webhook"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(iprewritev1alpha1.AddToScheme(scheme))
	utilruntime.Must(virtv1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool
	var certDir string
	var webhookPort int
	var initContainerImage string
	var initContainerPullPolicy string
	var requestKVMDevice bool
	var enableHTTP2 bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "Metrics endpoint bind address")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "Health probe bind address")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true, "Enable leader election for controller")
	flag.StringVar(&certDir, "cert-dir", "/tmp/k8s-webhook-server/serving-certs",
		"Directory with tls.crt/tls.key (OpenShift Service CA mounts here)")
	flag.IntVar(&webhookPort, "webhook-port", 9443, "Webhook listen port")
	flag.StringVar(&initContainerImage, "init-container-image", ipwebhook.DefaultInitContainerImage,
		"Image for the IP rewrite init container")
	flag.StringVar(&initContainerPullPolicy, "init-container-pull-policy", "IfNotPresent",
		"imagePullPolicy for the injected init container")
	flag.BoolVar(&requestKVMDevice, "request-kvm-device", true,
		"Request devices.kubevirt.io/kvm for the init container")
	flag.BoolVar(&enableHTTP2, "enable-http2", false, "Enable HTTP/2 on the webhook server")

	opts := zap.Options{Development: false}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var tlsOpts []func(*tls.Config)
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, func(c *tls.Config) {
			setupLog.Info("disabling HTTP/2")
			c.NextProtos = []string{"http/1.1"}
		})
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    webhookPort,
			CertDir: certDir,
			TLSOpts: tlsOpts,
		}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "ip-rewrite-operator.reip.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := (&controller.IPRewriteConfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IPRewriteConfig")
		os.Exit(1)
	}

	if err := (&controller.VMAppliedReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VMApplied")
		os.Exit(1)
	}

	handler := &ipwebhook.Handler{
		InitContainerImage:      initContainerImage,
		InitContainerPullPolicy: corev1.PullPolicy(initContainerPullPolicy),
		RequestKVMDevice:        requestKVMDevice,
	}
	mgr.GetWebhookServer().Register(ipwebhook.MutatePodPath, &webhook.Admission{Handler: handler})

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", mgr.GetWebhookServer().StartedChecker()); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager",
		"initContainerImage", initContainerImage,
		"requestKVMDevice", requestKVMDevice)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
