package foreigncluster

import (
	discoveryv1alpha1 "github.com/liqotech/liqo/apis/discovery/v1alpha1"
	"github.com/liqotech/liqo/pkg/discovery"
)

// GetDiscoveryType returns the discovery type for the given ForeignCluster.
func GetDiscoveryType(foreignCluster *discoveryv1alpha1.ForeignCluster) discovery.Type {
	labels := foreignCluster.GetLabels()
	if l, ok := labels[discovery.DiscoveryTypeLabel]; ok {
		return discovery.Type(l)
	}
	return discovery.ManualDiscovery
}

// SetDiscoveryType sets the discovery type to the given ForeignCluster.
func SetDiscoveryType(foreignCluster *discoveryv1alpha1.ForeignCluster, disocveryType discovery.Type) {
	labels := foreignCluster.GetLabels()
	labels[discovery.DiscoveryTypeLabel] = string(disocveryType)
	foreignCluster.SetLabels(labels)
}
