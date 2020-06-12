/*

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
	"errors"
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	protocolv1 "github.com/liqoTech/liqo/api/advertisement-operator/v1"
	liqonetv1 "github.com/liqoTech/liqo/api/tunnel-endpoint/v1"
	"github.com/liqoTech/liqo/internal/advertisement-operator"
	// +kubebuilder:scaffold:imports
)

const (
	defaultNamespace   = "default"
	defaultMetricsaddr = ":8080"
	defaultVKImage     = "liqo/virtual-kubelet"
	defaultInitVKImage = "liqo/init-vkubelet"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = protocolv1.AddToScheme(scheme)

	_ = liqonetv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr, localKubeconfig, foreignKubeconfig, clusterId string
	var enableLeaderElection bool
	var kubeletNamespace string
	var kubeletImage string
	var initKubeletImage string
	var runsInKindEnv bool

	flag.StringVar(&metricsAddr, "metrics-addr", defaultMetricsaddr, "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&localKubeconfig, "local-kubeconfig", "", "The path to the kubeconfig of your local cluster.")
	flag.StringVar(&foreignKubeconfig, "foreign-kubeconfig", "", "The path to the kubeconfig of the foreign cluster.")
	flag.StringVar(&clusterId, "cluster-id", "", "The cluster ID of your cluster")
	flag.StringVar(&kubeletNamespace, "kubelet-namespace", defaultNamespace, "Name of the namespace where Virtual kubelets will be spawned ( the namespace is default if not specified otherwise)")
	flag.StringVar(&kubeletImage, "kubelet-image", defaultVKImage, "The image of the virtual kubelet to be deployed")
	flag.StringVar(&initKubeletImage, "init-kubelet-image", defaultInitVKImage, "The image of the virtual kubelet init container to be deployed")
	flag.BoolVar(&runsInKindEnv, "run-in-kind", false, "The cluster in which the controller runs is managed by kind")
	flag.Parse()

	ctrl.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = true
	}))

	if clusterId == "" {
		setupLog.Error(errors.New("Cluster ID must be provided "), "")
		os.Exit(1)
	}

	if localKubeconfig != "" {
		if err := os.Setenv("KUBECONFIG", localKubeconfig); err != nil {
			os.Exit(1)
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
		LeaderElection:     enableLeaderElection,
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&advertisement_operator.AdvertisementReconciler{
		Client:           mgr.GetClient(),
		Log:              ctrl.Log.WithName("controllers").WithName("Advertisement"),
		Scheme:           mgr.GetScheme(),
		EventsRecorder:   mgr.GetEventRecorderFor("AdvertisementOperator"),
		KindEnvironment:  runsInKindEnv,
		KubeletNamespace: kubeletNamespace,
		VKImage:          kubeletImage,
		InitVKImage:      initKubeletImage,
		HomeClusterId:    clusterId,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Advertisement")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager as advertisement-operator")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

}
