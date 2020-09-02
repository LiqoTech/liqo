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

package v1alpha1

import (
	object_references "github.com/liqoTech/liqo/pkg/object-references"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkInfo struct {
	PodCIDR          string `json:"podCIDR"`
	GatewayIP        string `json:"gatewayIP"`
	GatewayPrivateIP string `json:"gatewayPrivateIP"`
	// +optional
	SupportedProtocols []string `json:"supportedProtocols,omitempty"`
}

type NamespacedName struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// AdvertisementSpec defines the desired state of Advertisement
type AdvertisementSpec struct {
	// ClusterId is the identifier of the cluster that is sending this Advertisement.
	// It is the uid of the first master node in you cluster.
	ClusterId string `json:"clusterId"`
	// Images is the list of the images already stored in the cluster.
	Images []corev1.ContainerImage `json:"images,omitempty"`
	// LimitRange contains the limits for every kind of resource (cpu, memory...).
	LimitRange corev1.LimitRangeSpec `json:"limitRange,omitempty"`
	// ResourceQuota contains the quantity of resources made available by the cluster.
	ResourceQuota corev1.ResourceQuotaSpec `json:"resourceQuota,omitempty"`
	// Neighbors is a map where the key is the name of a virtual node (representing a foreign cluster) and the value are the resources allocatable on that node.
	Neighbors map[corev1.ResourceName]corev1.ResourceList `json:"neighbors,omitempty"`
	// Properties can contain any additional information about the cluster.
	Properties map[corev1.ResourceName]string `json:"properties,omitempty"`
	// Prices contains the possible prices for every kind of resource (cpu, memory, image).
	Prices corev1.ResourceList `json:"prices,omitempty"`
	// Network contains the network information of the cluster.
	Network NetworkInfo `json:"network"`
	// KubeConfigRef is a reference to a secret containing the kubeconfig for the virtual-kubelet.
	// The virtual-kubelet will use this kubeconfig to access to the foreign cluster which is sending this Advertisement.
	KubeConfigRef corev1.SecretReference `json:"kubeConfigRef"`
	// Timestamp is the time instant when this Advertisement was created.
	Timestamp metav1.Time `json:"timestamp"`
	// TimeToLive is the time instant until this Advertisement will be valid.
	// If not refreshed, an Advertisement will expire after 30 minutes.
	TimeToLive metav1.Time `json:"timeToLive"`
}

// AdvPhase describes the phase of the Advertisement
type AdvPhase string

const (
	AdvertisementAccepted AdvPhase = "Accepted"
	AdvertisementRefused  AdvPhase = "Refused"
	AdvertisementDeleting AdvPhase = "Deleting"
)

// AdvertisementStatus defines the observed state of Advertisement
type AdvertisementStatus struct {
	// AdvertisementStatus is the status of this Advertisement.
	// When the adv is created it is checked by the operator, which sets this field to "Accepted" or "Refused" on tha base of cluster configuration.
	// If the Advertisement is accepted a virtual-kubelet for the foreign cluster will be created.
	// When the cluster wants to stop sharing its resources, it sets AdvertisementStatus to "Deleting" to allow the virtual-kubelet to delete the resources it had created,
	// then the Advertisement is deleted.
	// +kubebuilder:validation:Enum="";"Accepted";"Refused";"Deleting"
	AdvertisementStatus AdvPhase `json:"advertisementStatus"`
	// VkCreated indicates if the virtual-kubelet for this Advertisement has been created or not.
	VkCreated bool `json:"vkCreated"`
	// VkReference is a reference to the deployment running the virtual-kubelet.
	VkReference object_references.DeploymentReference `json:"vkReference,omitempty"`
	// LocalRemappedPodCIDR contains how the foreign cluster (sending the Advertisement) has remapped home cluster (receiving the Advertisement) pod CIDR.
	// If no overlapping occurred, this field is set to "None"
	LocalRemappedPodCIDR string `json:"localRemappedPodCIDR,omitempty"`
	// RemoteRemappedPodCIDR contains how the home cluster (receiving the Advertisement) has remapped foreign cluster (sendind the Advertisement) pod CIDR.
	// If no overlapping occurred, this field is set to "None"
	RemoteRemappedPodCIDR string `json:"remoteRemappedPodCIDR,omitempty"`
	// TunnelEndpointKey contains the namespaced name of the tunnelEndpoint associated with the foreign cluster
	TunnelEndpointKey NamespacedName `json:"tunnelEndpointKey"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName="adv"
// +kubebuilder:resource:scope=Cluster

// Advertisement is the Schema for the advertisements API
type Advertisement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AdvertisementSpec   `json:"spec,omitempty"`
	Status AdvertisementStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AdvertisementList contains a list of Advertisement
type AdvertisementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Advertisement `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Advertisement{}, &AdvertisementList{})
}
