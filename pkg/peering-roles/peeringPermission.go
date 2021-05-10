package peeringRoles

import (
	"context"
	"reflect"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"github.com/liqotech/liqo/pkg/auth"
)

type PeeringPermission struct {
	// to be enabled with the creation of the Tenant Control Namespace, these ClusterRoles have the basic permissions to give to a remote cluster
	Basic []*rbacv1.ClusterRole
	// to be enabled when a ResourceRequest has been accepted, these ClusterRoles have the permissions required to a remote cluster to manage an outgoing peering (incoming for the local cluster), when the Pods will be offloaded to the local cluster
	Incoming []*rbacv1.ClusterRole
	// to be enabled when we send a ResourceRequest, these ClusterRoles have the permissions required to a remote cluster to manage an incoming peering (outgoing for the local cluster), when the Pods will be offloaded from the local cluster
	Outgoing []*rbacv1.ClusterRole
}

// GetPeeringPermission populates a PeeringPermission with the ClusterRole names provided by the configuration.
func GetPeeringPermission(client kubernetes.Interface, config auth.AuthConfigProvider) (*PeeringPermission, error) {
	if config == nil || reflect.ValueOf(config).IsNil() {
		klog.Warning("no ClusterConfig set")
		return &PeeringPermission{}, nil
	}

	peeringPermission := config.GetConfig().PeeringPermission

	if peeringPermission == nil {
		klog.Warning("no peering permission set in the ClusterConfig CR")
		return &PeeringPermission{}, nil
	}

	basic, err := getClusterRoles(client, peeringPermission.Basic)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	incoming, err := getClusterRoles(client, peeringPermission.Incoming)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	outgoing, err := getClusterRoles(client, peeringPermission.Outgoing)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &PeeringPermission{
		Basic:    basic,
		Incoming: incoming,
		Outgoing: outgoing,
	}, nil
}

// getClusterRoles gets a ClusterRole given the name.
func getClusterRoles(client kubernetes.Interface, names []string) ([]*rbacv1.ClusterRole, error) {
	if names == nil {
		return []*rbacv1.ClusterRole{}, nil
	}

	var err error
	clusterRoles := make([]*rbacv1.ClusterRole, len(names))
	for i, name := range names {
		if clusterRoles[i], err = client.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{}); err != nil {
			klog.Error(err)
			return nil, err
		}
	}
	return clusterRoles, nil
}
