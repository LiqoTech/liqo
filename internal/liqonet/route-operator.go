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
package controllers

import (
	"context"
	"fmt"
	"github.com/coreos/go-iptables/iptables"
	"github.com/go-logr/logr"
	"github.com/liqoTech/liqo/api/tunnel-endpoint/v1"
	dronetOperator "github.com/liqoTech/liqo/pkg/liqonet"
	"github.com/vishvananda/netlink"
	k8sApiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strconv"
	"strings"
	"syscall"
)

var (
	dronetPostroutingChain = "DRONET-POSTROUTING"
	dronetPreroutingChain  = "DRONET-PREROUTING"
	dronetForwardingChain  = "DRONET-FORWARD"
	dronetInputChain       = "DRONET-INPUT"
	natTable               = "nat"
	filterTable            = "filter"
	shutdownSignals        = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGKILL}
)

// RouteController reconciles a TunnelEndpoint object
type RouteController struct {
	client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
	clientset      kubernetes.Clientset
	RouteOperator  bool
	NodeName       string
	ClientSet      *kubernetes.Clientset
	RemoteVTEPs    []string
	IsGateway      bool
	VxlanNetwork   string
	GatewayVxlanIP string
	VxlanIfaceName string
	VxlanPort      int
	ClusterPodCIDR string
	//here we save only the rules that reference the custom chains added by us
	//we need them at deletion time
	IPTablesRuleSpecsReferencingChains map[string]dronetOperator.IPtableRule //using a map to avoid duplicate entries. the key is the rulespec
	//here we save the custom iptables chains, this chains are added at startup time so there should not be duplicates
	//but we use a map to avoid them in case the operator crashes and then is restarted by kubernetes
	IPTablesChains map[string]dronetOperator.IPTableChain
	//for each cluster identified by clusterID we save all the rulespecs needed to ensure communication with its pods
	IPtablesRuleSpecsPerRemoteCluster map[string][]dronetOperator.IPtableRule
	//here we save routes associated to each remote cluster
	RoutesPerRemoteCluster map[string][]netlink.Route
}

// +kubebuilder:rbac:groups=dronet.drone.com,resources=tunnelendpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dronet.drone.com,resources=tunnelendpoints/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list

func (r *RouteController) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("endpoint", req.NamespacedName)
	var endpoint v1.TunnelEndpoint
	//name of our finalizer
	routeOperatorFinalizer := "routeOperator-" + r.NodeName + "-Finalizer.dronet.drone.com"

	if err := r.Get(ctx, req.NamespacedName, &endpoint); err != nil {
		r.Log.Error(err, "unable to fetch endpoint")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if endpoint.ObjectMeta.DeletionTimestamp.IsZero() {
		if !dronetOperator.ContainsString(endpoint.ObjectMeta.Finalizers, routeOperatorFinalizer) {
			// The object is not being deleted, so if it does not have our finalizer,
			// then lets add the finalizer and update the object. This is equivalent
			// registering our finalizer.
			endpoint.ObjectMeta.Finalizers = append(endpoint.Finalizers, routeOperatorFinalizer)
			if err := r.Update(ctx, &endpoint); err != nil {
				//while updating we check if the a resource version conflict happened
				//which means the version of the object we have is outdated.
				//a solution could be to return an error and requeue the object for later process
				//but if the object has been changed by another instance of the controller running in
				//another host it already has been put in the working queue so decide to forget the
				//current version and process the next item in the queue assured that we handle the object later
				if k8sApiErrors.IsConflict(err) {
					return ctrl.Result{}, nil
				}
				r.Log.Error(err, "unable to update endpoint")
				return ctrl.Result{}, err
			}
		}
	} else {
		//the object is being deleted
		if dronetOperator.ContainsString(endpoint.Finalizers, routeOperatorFinalizer) {
			if err := r.deleteIPTablesRulespecForRemoteCluster(&endpoint); err != nil {
				r.Log.Error(err, "error while deleting rulespec from iptables")
				return ctrl.Result{}, err
			}
			if err := r.deleteRoutesPerCluster(&endpoint); err != nil {
				r.Log.Error(err, "error while deleting routes")
				return ctrl.Result{}, err
			}
			//remove the finalizer from the list and update it.
			endpoint.Finalizers = dronetOperator.RemoveString(endpoint.Finalizers, routeOperatorFinalizer)
			if err := r.Update(ctx, &endpoint); err != nil {
				if k8sApiErrors.IsConflict(err) {
					return ctrl.Result{}, nil
				}
				r.Log.Error(err, "unable to update")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	if !r.alreadyProcessedByRouteOperator(endpoint.GetObjectMeta()) {
		if err := r.createAndInsertIPTablesChains(); err != nil {
			r.Log.Error(err, "unable to create iptables chains")
			return ctrl.Result{}, err
		}
		if err := r.addIPTablesRulespecForRemoteCluster(&endpoint); err != nil {
			log.Error(err, "unable to insert ruleSpec")
			return ctrl.Result{}, err
		}
		if err := r.InsertRoutesPerCluster(&endpoint); err != nil {
			log.Error(err, "unable to insert routes")
			return ctrl.Result{}, err
		}
		endpoint.ObjectMeta.SetLabels(dronetOperator.SetLabelHandler(dronetOperator.RouteOpLabelKey+"-"+r.NodeName, "ready", endpoint.ObjectMeta.GetLabels()))
		err := r.Client.Update(ctx, &endpoint)
		for k8sApiErrors.IsConflict(err) {
			log.Info("a resource version conflict arose while updating", "resource", req.NamespacedName)
			if err := r.Get(ctx, req.NamespacedName, &endpoint); err != nil {
				r.Log.Error(err, "unable to fetch endpoint")
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
			endpoint.ObjectMeta.SetLabels(dronetOperator.SetLabelHandler(dronetOperator.RouteOpLabelKey+"-"+r.NodeName, "ready", endpoint.ObjectMeta.GetLabels()))
			err = r.Client.Update(ctx, &endpoint)
		}
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

//this function is called at startup of the operator
//here we:
//create DRONET-FORWARD in the filter table and insert it in the "FORWARD" chain
//create DRONET-POSTROUTING in the nat table and insert it in the "POSTROUTING" chain
//create DRONET-INPUT in the filter table and insert it in the input chain
//insert the rulespec which allows in input all the udp traffic incoming for the vxlan in the DRONET-INPUT chain
func (r *RouteController) createAndInsertIPTablesChains() error {
	ipt, err := iptables.New()
	log := r.Log.WithName("iptables")
	if err != nil {
		return fmt.Errorf("unable to initialize iptables: %v. check if the ipatable are present in the system", err)
	}

	//creating DRONET-POSTROUTING chain
	if err = dronetOperator.CreateIptablesChainsIfNotExist(ipt, natTable, dronetPostroutingChain); err != nil {
		return err
	} else {
		log.Info("created", "chain", dronetPostroutingChain, "in table", natTable)
	}
	r.IPTablesChains[dronetPostroutingChain] = dronetOperator.IPTableChain{
		Table: natTable,
		Name:  dronetPostroutingChain,
	}
	//installing rulespec which forwards all traffic to DRONET-POSTROUTING chain
	forwardToDronetPostroutingRuleSpec := []string{"-j", dronetPostroutingChain}
	if err = dronetOperator.InsertIptablesRulespecIfNotExists(ipt, natTable, "POSTROUTING", forwardToDronetPostroutingRuleSpec); err != nil {
		return err
	} else {
		log.Info("installed", "rulespec", strings.Join(forwardToDronetPostroutingRuleSpec, " "), "belonging to chain POSTROUTING in table", natTable)
	}
	//add it to iptables rulespec if it does not exist in the map
	r.IPTablesRuleSpecsReferencingChains[strings.Join(forwardToDronetPostroutingRuleSpec, " ")] = dronetOperator.IPtableRule{
		Table:    natTable,
		Chain:    "POSTROUTING",
		RuleSpec: forwardToDronetPostroutingRuleSpec,
	}
	//creating DRONET-PREROUTING chain
	if err = dronetOperator.CreateIptablesChainsIfNotExist(ipt, natTable, dronetPreroutingChain); err != nil {
		return err
	} else {
		log.Info("created", "chain", dronetPreroutingChain, "in table", natTable)
	}
	r.IPTablesChains[dronetPostroutingChain] = dronetOperator.IPTableChain{
		Table: natTable,
		Name:  dronetPreroutingChain,
	}
	//installing rulespec which forwards all traffic to DRONET-PREROUTING chain
	forwardToDronetPreroutingRuleSpec := []string{"-j", dronetPreroutingChain}
	if err = dronetOperator.InsertIptablesRulespecIfNotExists(ipt, natTable, "PREROUTING", forwardToDronetPreroutingRuleSpec); err != nil {
		return err
	} else {
		log.Info("installed", "rulespec", strings.Join(forwardToDronetPreroutingRuleSpec, " "), "belonging to chain POSTROUTING in table", natTable)
	}
	//add it to iptables rulespec if it does not exist in the map
	r.IPTablesRuleSpecsReferencingChains[strings.Join(forwardToDronetPreroutingRuleSpec, " ")] = dronetOperator.IPtableRule{
		Table:    natTable,
		Chain:    "PREROUTING",
		RuleSpec: forwardToDronetPreroutingRuleSpec,
	}
	//creating DRONET-FORWARD chain
	if err = dronetOperator.CreateIptablesChainsIfNotExist(ipt, filterTable, dronetForwardingChain); err != nil {
		return err
	} else {
		log.Info("created", "chain", dronetForwardingChain, "in table", filterTable)
	}
	r.IPTablesChains[dronetForwardingChain] = dronetOperator.IPTableChain{
		Table: filterTable,
		Name:  dronetForwardingChain,
	}
	//installing rulespec which forwards all traffic to DRONET-FORWARD chain
	forwardToDronetForwardRuleSpec := []string{"-j", dronetForwardingChain}
	if err = dronetOperator.InsertIptablesRulespecIfNotExists(ipt, filterTable, "FORWARD", forwardToDronetForwardRuleSpec); err != nil {
		return err
	} else {
		log.Info("installed", "rulespec", strings.Join(forwardToDronetForwardRuleSpec, " "), "belonging to chain FORWARD in table", filterTable)
	}
	r.IPTablesRuleSpecsReferencingChains[strings.Join(forwardToDronetForwardRuleSpec, " ")] = dronetOperator.IPtableRule{
		Table:    filterTable,
		Chain:    "FORWARD",
		RuleSpec: forwardToDronetForwardRuleSpec,
	}
	//creating DRONET-INPUT chain
	if err = dronetOperator.CreateIptablesChainsIfNotExist(ipt, filterTable, dronetInputChain); err != nil {
		return err
	} else {
		log.Info("created", "chain", dronetInputChain, "in table", filterTable)
	}
	r.IPTablesChains[dronetInputChain] = dronetOperator.IPTableChain{
		Table: filterTable,
		Name:  dronetInputChain,
	}
	//installing rulespec which forwards all udp incoming traffic to DRONET-INPUT chain
	forwardToDronetInputSpec := []string{"-p", "udp", "-m", "udp", "-j", dronetInputChain}
	if err = dronetOperator.InsertIptablesRulespecIfNotExists(ipt, filterTable, "INPUT", forwardToDronetInputSpec); err != nil {
		return err
	} else {
		log.Info("installed", "rulespec", strings.Join(forwardToDronetInputSpec, " "), "belonging to chain POSTROUTING in table", filterTable)
	}
	r.IPTablesRuleSpecsReferencingChains[strings.Join(forwardToDronetInputSpec, " ")] = dronetOperator.IPtableRule{
		Table:    filterTable,
		Chain:    "INPUT",
		RuleSpec: forwardToDronetInputSpec,
	}
	//installing rulespec which allows udp traffic with destination port the VXLAN port
	//we put it here because this rulespec is independent from the remote cluster.
	//we don't save this rulespec it will be removed when the chains are flushed at exit time
	//TODO: do we need to move this one elsewhere? maybe in a dedicate function called at startup by the route operator?
	vxlanUdpRuleSpec := []string{"-p", "udp", "-m", "udp", "--dport", strconv.Itoa(r.VxlanPort), "-j", "ACCEPT"}
	if err = ipt.AppendUnique(filterTable, dronetInputChain, vxlanUdpRuleSpec...); err != nil {
		return fmt.Errorf("unable to insert rulespec \"%s\" in %s table and %s chain: %v", vxlanUdpRuleSpec, filterTable, dronetInputChain, err)
	} else {
		log.Info("installed", "rulespec", strings.Join(vxlanUdpRuleSpec, " "), "belonging to chain", dronetInputChain, "in table", filterTable)
	}
	return nil
}

func (r *RouteController) addIPTablesRulespecForRemoteCluster(endpoint *v1.TunnelEndpoint) error {
	var remotePodCIDR string
	clusterID := endpoint.Spec.ClusterID
	log := r.Log.WithName("iptables")
	if endpoint.Status.RemoteRemappedPodCIDR != "None" && endpoint.Status.RemoteRemappedPodCIDR != "" {
		remotePodCIDR = endpoint.Status.RemoteRemappedPodCIDR
		log.Info("nat enabled", "pod cidr of cluster", clusterID, "remapped from", endpoint.Spec.PodCIDR, "to", endpoint.Status.RemoteRemappedPodCIDR)
	} else {
		remotePodCIDR = endpoint.Spec.PodCIDR
		log.Info("nat disabled", "using original pod cidr", endpoint.Spec.PodCIDR, "for cluster", clusterID)
	}
	var ruleSpecs []dronetOperator.IPtableRule
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("unable to initialize iptables: %v. check if the ipatable are present in the system", err)
	}

	//do not nat the traffic directed to the remote pods
	ruleSpec := []string{"-s", r.ClusterPodCIDR, "-d", remotePodCIDR, "-j", "ACCEPT"}
	if err = ipt.AppendUnique(natTable, dronetPostroutingChain, ruleSpec...); err != nil {
		return fmt.Errorf("unable to insert iptable rule \"%s\" in %s table, %s chain: %v", ruleSpec, natTable, dronetPostroutingChain, err)
	} else {
		log.Info("installed", "rulespec", strings.Join(ruleSpec, " "), "belonging to chain", dronetPostroutingChain, "in table", filterTable)
	}
	ruleSpecs = append(ruleSpecs, dronetOperator.IPtableRule{
		Table:    natTable,
		Chain:    dronetPostroutingChain,
		RuleSpec: ruleSpec,
	})
	r.IPtablesRuleSpecsPerRemoteCluster[clusterID] = ruleSpecs
	//enable forwarding for all the traffic directed to the remote pods
	ruleSpec = []string{"-d", remotePodCIDR, "-j", "ACCEPT"}
	if err = ipt.AppendUnique(filterTable, dronetForwardingChain, ruleSpec...); err != nil {
		return fmt.Errorf("unable to insert iptable rule \"%s\" in %s table, %s chain: %v", ruleSpec, filterTable, dronetForwardingChain, err)
	} else {
		log.Info("installed", "rulespec", strings.Join(ruleSpec, " "), "belonging to chain", dronetForwardingChain, "in table", filterTable)
	}
	ruleSpecs = append(ruleSpecs, dronetOperator.IPtableRule{
		Table:    filterTable,
		Chain:    dronetForwardingChain,
		RuleSpec: ruleSpec,
	})
	r.IPtablesRuleSpecsPerRemoteCluster[clusterID] = ruleSpecs
	//this rules are needed in an environment where strictly policies are applied for the input chain
	ruleSpec = []string{"-s", r.ClusterPodCIDR, "-d", remotePodCIDR, "-j", "ACCEPT"}
	if err = ipt.AppendUnique(filterTable, dronetInputChain, ruleSpec...); err != nil {
		return fmt.Errorf("unable to insert iptable rule \"%s\" in %s table, %s chain: %v", ruleSpec, filterTable, dronetInputChain, err)
	} else {
		log.Info("installed", "rulespec", strings.Join(ruleSpec, " "), "belonging to chain", dronetInputChain, "in table", filterTable)
	}
	ruleSpecs = append(ruleSpecs, dronetOperator.IPtableRule{
		Table:    filterTable,
		Chain:    dronetInputChain,
		RuleSpec: ruleSpec,
	})
	r.IPtablesRuleSpecsPerRemoteCluster[clusterID] = ruleSpecs
	if r.IsGateway {
		//all the traffic coming from the hosts and directed to the remote pods is natted using the LocalTunnelPrivateIP
		//hosts use the ip of the vxlan interface as source ip when communicating with remote pods
		//this is done on the gateway node only
		ruleSpec = []string{"-s", r.VxlanNetwork, "-d", remotePodCIDR, "-j", "MASQUERADE"}
		if err = ipt.AppendUnique(natTable, dronetPostroutingChain, ruleSpec...); err != nil {
			return fmt.Errorf("unable to insert iptable rule \"%s\" in %s table, %s chain: %v", ruleSpec, natTable, dronetPostroutingChain, err)
		} else {
			log.Info("installed", "rulespec", strings.Join(ruleSpec, " "), "belonging to chain", dronetPostroutingChain, "in table", natTable)
		}
		ruleSpecs = append(ruleSpecs, dronetOperator.IPtableRule{
			Table:    natTable,
			Chain:    dronetPostroutingChain,
			RuleSpec: ruleSpec,
		})
		r.IPtablesRuleSpecsPerRemoteCluster[clusterID] = ruleSpecs
		//if we have been remapped by the remote cluster then insert the iptables rule to masquerade the source ip
		if endpoint.Status.LocalRemappedPodCIDR != "None" {
			ruleSpec = []string{"-s", r.ClusterPodCIDR, "-d", remotePodCIDR, "-j", "NETMAP", "--to", endpoint.Status.LocalRemappedPodCIDR}
			if err = dronetOperator.InsertIptablesRulespecIfNotExists(ipt, natTable, dronetPostroutingChain, ruleSpec); err != nil {
				return fmt.Errorf("unable to insert iptable rule \"%s\" in %s table, %s chain: %v", ruleSpec, natTable, dronetPostroutingChain, err)
			} else {
				log.Info("installed", "rulespec", strings.Join(ruleSpec, " "), "belonging to chain", dronetPostroutingChain, "in table", natTable)
			}
			ruleSpecs = append(ruleSpecs, dronetOperator.IPtableRule{
				Table:    natTable,
				Chain:    dronetPostroutingChain,
				RuleSpec: ruleSpec,
			})
			r.IPtablesRuleSpecsPerRemoteCluster[clusterID] = ruleSpecs
			//translate all the traffic coming to the local cluster in to the right podcidr because it has been remapped by the remote cluster
			ruleSpec = []string{"-d", endpoint.Status.LocalRemappedPodCIDR, "-i", endpoint.Status.TunnelIFaceName, "-j", "NETMAP", "--to", r.ClusterPodCIDR}
			if err = ipt.AppendUnique(natTable, dronetPreroutingChain, ruleSpec...); err != nil {
				return fmt.Errorf("unable to insert iptable rule \"%s\" in %s table, %s chain: %v", ruleSpec, natTable, dronetPreroutingChain, err)
			} else {
				log.Info("installed", "rulespec", strings.Join(ruleSpec, " "), "belonging to chain", dronetPreroutingChain, "in table", natTable)
			}
			ruleSpecs = append(ruleSpecs, dronetOperator.IPtableRule{
				Table:    natTable,
				Chain:    dronetPreroutingChain,
				RuleSpec: ruleSpec,
			})
			r.IPtablesRuleSpecsPerRemoteCluster[clusterID] = ruleSpecs
		}
	}
	return nil
}

//remove all the rules added by addIPTablesRulespecForRemoteCluster function
func (r *RouteController) deleteIPTablesRulespecForRemoteCluster(endpoint *v1.TunnelEndpoint) error {
	clusterID := endpoint.Spec.ClusterID
	log := r.Log.WithName("iptables")
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("unable to initialize iptables: %v. check if the ipatable are present in the system", err)
	}
	//retrive the iptables rules for the remote cluster
	rules, ok := r.IPtablesRuleSpecsPerRemoteCluster[endpoint.Spec.ClusterID]
	if ok {
		for _, rule := range rules {
			if err = ipt.Delete(rule.Table, rule.Chain, rule.RuleSpec...); err != nil {
				// if the rule that we are trying to delete does not exist then we are fine and go on
				e, ok := err.(*iptables.Error)
				if ok && e.IsNotExist() {
					continue
				} else if !ok {
					return fmt.Errorf("unable to delete iptable rule \"%s\" in %s table, %s chain: %v", strings.Join(rule.RuleSpec, " "), rule.Table, rule.Chain, err)
				}
			} else {
				log.Info("removing", "rulespec", strings.Join(rule.RuleSpec, " "), "belonging to chain", rule.Chain, "in table", rule.Table)
			}
		}
	}
	//after all the iptables rules have been removed then we delete them from the map
	//this is safe to do even if the key does not exist
	delete(r.IPtablesRuleSpecsPerRemoteCluster, clusterID)
	return nil
}

//this function is called when the route-operator program is closed
//the errors are not checked because the function is called at exit time
//it cleans up all the possible resources
//a log message is emitted if in case of error
//only if the iptables binaries are missing an error is returned
func (r *RouteController) DeleteAllIPTablesChains() {
	logger := r.Log.WithName("DeleteAllIPTablesChains")
	ipt, err := iptables.New()
	if err != nil {
		logger.Error(err, "unable to initialize iptables.check if the ipatable are present in the system")
	}
	//first all the iptables chains are flushed
	for _, chain := range r.IPTablesChains {
		if err = ipt.ClearChain(chain.Table, chain.Name); err != nil {
			e, ok := err.(*iptables.Error)
			if ok && e.IsNotExist() {
				continue
			} else if !ok {
				logger.Error(err, "unable to clear: ", "chain", chain.Name, "in table", chain.Table)
			}

		}
	}
	//second we delete the references to the chains
	for _, rulespec := range r.IPTablesRuleSpecsReferencingChains {
		if err = ipt.Delete(rulespec.Table, rulespec.Chain, rulespec.RuleSpec...); err != nil {
			e, ok := err.(*iptables.Error)
			if ok && e.IsNotExist() {
				continue
			} else if !ok {
				logger.Error(err, "unable to delete: ", "rule", strings.Join(rulespec.RuleSpec, ""), "in chain", rulespec.Chain, "in table", rulespec.Table)
			}
		}
	}
	//then we delete the chains which now should be empty
	for _, chain := range r.IPTablesChains {
		if err = ipt.DeleteChain(chain.Table, chain.Name); err != nil {
			e, ok := err.(*iptables.Error)
			if ok && e.IsNotExist() {
				continue
			} else if !ok {
				logger.Error(err, "unable to delete ", "chain", chain.Name, "in table", chain.Table)
			}
		}
	}
}

func (r *RouteController) InsertRoutesPerCluster(endpoint *v1.TunnelEndpoint) error {
	clusterID := endpoint.Spec.ClusterID
	log := r.Log.WithName("route")
	remoteTunnelPrivateIPNet := endpoint.Status.RemoteTunnelPrivateIP + "/32"
	var remotePodCIDR string
	localTunnelPrivateIP := endpoint.Status.LocalTunnelPrivateIP
	if endpoint.Status.RemoteRemappedPodCIDR != "None" && endpoint.Status.RemoteRemappedPodCIDR != "" {
		remotePodCIDR = endpoint.Status.RemoteRemappedPodCIDR
		log.Info("installing routes for", "cluster", clusterID, "with remapped pod cidr", remotePodCIDR)
	} else {
		remotePodCIDR = endpoint.Spec.PodCIDR
		log.Info("installing routes for", "cluster", clusterID, "with original pod cidr", remotePodCIDR)
	}
	var routes []netlink.Route
	if r.IsGateway {
		route, err := dronetOperator.AddRoute(remoteTunnelPrivateIPNet, localTunnelPrivateIP, endpoint.Status.TunnelIFaceName, false)
		if err != nil {
			return err
		} else {
			log.Info("installing", "route", route.String())
		}
		routes = append(routes, route)
		route, err = dronetOperator.AddRoute(remotePodCIDR, endpoint.Status.RemoteTunnelPrivateIP, endpoint.Status.TunnelIFaceName, true)
		if err != nil {
			return err
		} else {
			log.Info("installing", "route", route.String())
		}
		routes = append(routes, route)
		r.RoutesPerRemoteCluster[endpoint.Spec.ClusterID] = routes
	} else {
		route, err := dronetOperator.AddRoute(remotePodCIDR, r.GatewayVxlanIP, r.VxlanIfaceName, false)
		if err != nil {
			return err
		} else {
			log.Info("installing", "route", route.String())
		}
		routes = append(routes, route)
		route, err = dronetOperator.AddRoute(remoteTunnelPrivateIPNet, r.GatewayVxlanIP, r.VxlanIfaceName, false)
		if err != nil {
			return err
		} else {
			log.Info("installing", "route", route.String())
		}
		routes = append(routes, route)
		r.RoutesPerRemoteCluster[endpoint.Spec.ClusterID] = routes
	}
	return nil
}

//used to remove the routes when a tunnelEndpoint CR is removed
func (r *RouteController) deleteRoutesPerCluster(endpoint *v1.TunnelEndpoint) error {
	clusterID := endpoint.Spec.ClusterID
	log := r.Log.WithName("route")
	log.Info("removing all routes for", "cluster", clusterID)
	for _, route := range r.RoutesPerRemoteCluster[endpoint.Spec.ClusterID] {
		err := dronetOperator.DelRoute(route)
		if err != nil {
			return err
		} else {
			log.Info("deleting", "route", route.String())
		}
	}
	//after all the routes have been removed then we delete them from the map
	//this is safe to do even if the key does not exist
	delete(r.RoutesPerRemoteCluster, clusterID)
	return nil
}

func (r *RouteController) deleteAllRoutes() {
	logger := r.Log.WithName("DeleteAllRoutes")
	//the errors are not checked because the function is called at exit time
	//it cleans up all the possible resources
	//a log message is emitted if in case of error
	for _, cluster := range r.RoutesPerRemoteCluster {
		for _, route := range cluster {
			if err := dronetOperator.DelRoute(route); err != nil {
				logger.Error(err, "an error occurred while deleting", "route", route.String())
			}
		}
	}
}

//this function deletes the vxlan interface in host where the route operator is running
func (r *RouteController) deleteVxlanIFace() {
	logger := r.Log.WithName("DeleteVxlanIFace")
	//first get the iface index
	iface, err := netlink.LinkByName(r.VxlanIfaceName)
	if err != nil {
		logger.Error(err, "an error occurred while removing vxlan interface", "ifaceName", r.VxlanIfaceName)
	}
	err = dronetOperator.DeleteIFaceByIndex(iface.Attrs().Index)
	if err != nil {
		logger.Error(err, "an error occurred while removing vxlan interface", "ifaceName", r.VxlanIfaceName)
	}
}

// SetupSignalHandlerForRouteOperator registers for SIGTERM, SIGINT. A stop channel is returned
// which is closed on one of these signals.
func (r *RouteController) SetupSignalHandlerForRouteOperator() (stopCh <-chan struct{}) {
	logger := r.Log.WithValues("Route Operator Signal Handler", r.NodeName)
	fmt.Printf("Entering signal handler")
	stop := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, shutdownSignals...)
	go func(r *RouteController) {
		sig := <-c
		logger.Info("received ", "signal", sig.String())
		r.DeleteAllIPTablesChains()
		r.deleteAllRoutes()
		r.deleteVxlanIFace()
		<-c
		close(stop)
	}(r)
	return stop
}

//checks if all the values need to install routes have ben set in the CR status
func (r *RouteController) ValidateCRAndReturn(endpoint *v1.TunnelEndpoint) bool {
	isReady := true
	//check if the tunnel interface is installed but only if the route operator is running on the gatewayhost
	if endpoint.Status.TunnelIFaceIndex != 0 && r.IsGateway {
		_, err := netlink.LinkByIndex(endpoint.Status.TunnelIFaceIndex)
		if err != nil {
			isReady = false
		}
	}
	if endpoint.Status.RemoteTunnelPrivateIP == "" || endpoint.Status.RemoteTunnelPublicIP == "" {
		isReady = false
	}
	if endpoint.Status.LocalTunnelPrivateIP == "" || endpoint.Status.LocalTunnelPublicIP == "" {
		isReady = false
	}
	return isReady
}
func (r *RouteController) SetupWithManager(mgr ctrl.Manager) error {
	resourceToBeProccesedPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return r.ToBeProcessedByRouteOperator(e.Meta)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			//finalizers are used to check if a resource is being deleted, and perform there the needed actions
			//we don't want to reconcile on the delete of a resource.
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return r.ToBeProcessedByRouteOperator(e.MetaNew)
		},
	}
	return ctrl.NewControllerManagedBy(mgr).WithEventFilter(resourceToBeProccesedPredicate).
		For(&v1.TunnelEndpoint{}).
		Complete(r)
}

func (r *RouteController) ToBeProcessedByRouteOperator(meta metav1.Object) bool {
	labels := meta.GetLabels()
	if labels == nil {
		return false
	}
	_, processedByTunOP := labels[dronetOperator.TunOpLabelKey]
	if processedByTunOP {
		return true
	} else {
		return false
	}
}

func (r *RouteController) alreadyProcessedByRouteOperator(meta metav1.Object) bool {
	labels := meta.GetLabels()
	if labels == nil {
		return true
	}
	_, processedByRouOp := labels[dronetOperator.RouteOpLabelKey+"-"+r.NodeName]
	if processedByRouOp {
		return true
	} else {
		return false
	}
}
