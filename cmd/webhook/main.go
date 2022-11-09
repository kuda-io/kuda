/*
Copyright 2021 The Kuda Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	datav1alpha1 "github.com/kuda-io/kuda/pkg/api/data/v1alpha1"
	webhook2 "github.com/kuda-io/kuda/pkg/webhook"
)

var (
	scheme = runtime.NewScheme()
	log    = ctrl.Log.WithName("webhook")
)

func init() {
	_ = datav1alpha1.AddToScheme(scheme)
}

func main() {
	var (
		port       int
		certDir    string
		webhookCfg string
		probeAddr  string
	)
	flag.IntVar(&port, "port", 8443, "Port is the port that the webhook server serves at.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&certDir, "certDir", "/etc/webhook/certs", "CertDir is the directory that contains the server key and certificate.")
	flag.StringVar(&webhookCfg, "config", "/etc/webhook/config.yaml", "Config file path for the admission webhook.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	flag.Parse()

	ctrllog.SetLogger(zap.New())

	// setup manager
	mgr, err := ctrl.NewManager(config.GetConfigOrDie(), manager.Options{
		Scheme:                 scheme,
		Port:                   port,
		CertDir:                certDir,
		HealthProbeBindAddress: probeAddr,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// load config
	config, err := webhook2.LoadConfig(webhookCfg)
	if err != nil {
		log.Error(err, "unable to load config")
		os.Exit(1)
	}

	// setup webhook
	log.Info("setting up webhook server")
	ws := mgr.GetWebhookServer()
	ws.Register("/inject", &webhook.Admission{Handler: webhook2.NewPodInjector(config, mgr.GetClient())})

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}
}
