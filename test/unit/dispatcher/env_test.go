package dispatcher

import (
	"context"
	"fmt"
	policyv1 "github.com/liqoTech/liqo/api/cluster-config/v1"
	discoveryv1 "github.com/liqoTech/liqo/api/discovery/v1"
	"github.com/liqoTech/liqo/internal/dispatcher"
	"github.com/liqoTech/liqo/pkg/crdClient"
	"github.com/liqoTech/liqo/pkg/liqonet"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/scheme"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"testing"
	"time"
)

var (
	numberPeeringClusters = 1

	peeringIDTemplate         = "peering-cluster-"
	localClusterID            = "localClusterID"
	peeringClustersTestEnvs   = map[string]*envtest.Environment{}
	peeringClustersManagers   = map[string]ctrl.Manager{}
	peeringClustersDynClients = map[string]dynamic.Interface{}
	configClusterClient       *crdClient.CRDClient
	k8sManagerLocal           ctrl.Manager
	testEnvLocal              *envtest.Environment
	dOperator                 *dispatcher.DispatcherReconciler
)

func TestMain(m *testing.M) {
	setupEnv()
	defer tearDown()
	startDispatcherOperator()
	time.Sleep(10 * time.Second)
	klog.Info("main set up")
	os.Exit(m.Run())
}

func startDispatcherOperator() {
	err := setupDispatcherOperator()
	if err != nil {
		klog.Error(err)
		os.Exit(-1)
	}
	cacheStartedLocal := make(chan struct{})
	go func() {
		if err = k8sManagerLocal.Start(ctrl.SetupSignalHandler()); err != nil {
			klog.Error(err)
			panic(err)
		}
	}()
	started := k8sManagerLocal.GetCache().WaitForCacheSync(cacheStartedLocal)
	if !started {
		klog.Errorf("an error occurred while waiting for the chache to start")
		os.Exit(-1)
	}
	configLocal := k8sManagerLocal.GetConfig()
	newConfig := &rest.Config{
		Host: configLocal.Host,
		// gotta go fast during tests -- we don't really care about overwhelming our test API server
		QPS:   1000.0,
		Burst: 2000.0,
	}
	err = dOperator.WatchConfiguration(newConfig, &policyv1.GroupVersion)
	if err != nil {
		klog.Errorf("an error occurred while starting the configuration watcher of dispatcher operator: %s", err)
		os.Exit(-1)
	}
	fc := getForeignClusterResource()
	_, err = dOperator.LocalDynClient.Resource(fcGVR).Create(context.TODO(), fc, metav1.CreateOptions{})
	if err != nil {
		klog.Error(err, err.Error())
		os.Exit(-1)
	}
}

func getConfigClusterCRDClient(config *rest.Config) *crdClient.CRDClient {
	newConfig := config
	newConfig.ContentConfig.GroupVersion = &policyv1.GroupVersion
	newConfig.APIPath = "/apis"
	newConfig.NegotiatedSerializer = clientgoscheme.Codecs.WithoutConversion()
	newConfig.UserAgent = rest.DefaultKubernetesUserAgent()
	CRDclient, err := crdClient.NewFromConfig(newConfig)
	if err != nil {
		klog.Error(err, err.Error())
		os.Exit(1)
	}
	return CRDclient
}

func setupEnv() {
	err := discoveryv1.AddToScheme(scheme.Scheme)
	if err != nil {
		klog.Error(err)
	}
	//save the environment variables in the map
	for i := 1; i <= numberPeeringClusters; i++ {
		peeringClusterID := peeringIDTemplate + fmt.Sprintf("%d", i)
		peeringClustersTestEnvs[peeringClusterID] = &envtest.Environment{
			CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "deployments", "liqo_chart", "crds")},
		}
	}
	//start the peering environments, save the managers, create dynamic clients
	for peeringClusterID, testEnv := range peeringClustersTestEnvs {
		config, err := testEnv.Start()
		if err != nil {
			klog.Errorf("%s -> an error occurred while setting test environment: %s", peeringClusterID, err)
			os.Exit(-1)
		} else {
			klog.Infof("%s -> created test environment with configCluster %s", peeringClusterID, config.String())
		}
		manager, err := ctrl.NewManager(config, ctrl.Options{
			Scheme:             scheme.Scheme,
			MetricsBindAddress: "0",
		})
		if err != nil {
			klog.Errorf("%s -> an error occurred while creating the manager %s", peeringClusterID, err)
			os.Exit(-1)
		}
		peeringClustersManagers[peeringClusterID] = manager
		dynClient := dynamic.NewForConfigOrDie(manager.GetConfig())
		peeringClustersDynClients[peeringClusterID] = dynClient
	}
	//setup the local testing environment
	testEnvLocal = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "deployments", "liqo_chart", "crds")},
	}
	configLocal, err := testEnvLocal.Start()
	if err != nil {
		klog.Error(err, "an error occurred while setting up the local testing environment")
	}
	klog.Infof("%s -> created test environment with configCluster %s", localClusterID, configLocal.String())
	newConfig := &rest.Config{
		Host: configLocal.Host,
		// gotta go fast during tests -- we don't really care about overwhelming our test API server
		QPS:   1000.0,
		Burst: 2000.0,
	}
	k8sManagerLocal, err = ctrl.NewManager(configLocal, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",
	})
	if err != nil {
		klog.Errorf("%s -> an error occurred while creating the manager %s", localClusterID, err)
		os.Exit(-1)
	}
	configClusterClient = getConfigClusterCRDClient(newConfig)
	cc := getClusterConfig()
	_, err = configClusterClient.Resource("clusterconfigs").Create(cc, metav1.CreateOptions{})
	if err != nil {
		klog.Error(err, err.Error())
		os.Exit(-1)
	}
	klog.Info("setup of testing environments finished")
}

func tearDown() {
	//stop the peering testing environments
	for id, env := range peeringClustersTestEnvs {
		err := env.Stop()
		if err != nil {
			klog.Errorf("%s -> an error occurred while stopping peering environment test: %s", id, err)
		}
	}
	err := testEnvLocal.Stop()
	if err != nil {
		klog.Errorf("%s -> an error occurred while stopping local environment test: %s", localClusterID, err)
	}
}

func getClusterConfig() *policyv1.ClusterConfig {
	return &policyv1.ClusterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "configuration",
		},
		Spec: policyv1.ClusterConfigSpec{
			AdvertisementConfig: policyv1.AdvertisementConfig{
				IngoingConfig: policyv1.AdvOperatorConfig{
					AcceptPolicy:               policyv1.AutoAcceptWithinMaximum,
					MaxAcceptableAdvertisement: 5,
				},
				OutgoingConfig: policyv1.BroadcasterConfig{
					ResourceSharingPercentage: 30,
					EnableBroadcaster:         true,
				},
			},
			DiscoveryConfig: policyv1.DiscoveryConfig{
				AutoJoin:            true,
				Domain:              "local.",
				EnableAdvertisement: true,
				EnableDiscovery:     true,
				Name:                "MyLiqo",
				Port:                6443,
				Service:             "_liqo._tcp",
				UpdateTime:          3,
				WaitTime:            2,
				DnsServer:           "8.8.8.8:53",
			},
			LiqonetConfig: policyv1.LiqonetConfig{
				ReservedSubnets: []string{"10.0.0.0/16"},
				VxlanNetConfig: liqonet.VxlanNetConfig{
					Network:    "",
					DeviceName: "",
					Port:       "",
					Vni:        "",
				},
			},
			DispatcherConfig: policyv1.DispatcherConfig{ResourcesToReplicate: []policyv1.Resource{{
				Group:    "liqonet.liqo.io",
				Version:  "v1",
				Resource: "tunnelendpoints",
			}}},
		},
	}
}
