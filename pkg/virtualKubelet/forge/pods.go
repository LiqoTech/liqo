package forge

import (
	"fmt"
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"

	"github.com/liqotech/liqo/internal/liqonet/tunnelEndpointCreator"
	liqoconst "github.com/liqotech/liqo/pkg/consts"
	"github.com/liqotech/liqo/pkg/virtualKubelet"
	apimgmt "github.com/liqotech/liqo/pkg/virtualKubelet/apiReflection"
	"github.com/liqotech/liqo/pkg/virtualKubelet/apiReflection/reflectors"
)

const affinitySelector = liqoconst.TypeNode

func (f *apiForger) podForeignToHome(foreignObj, homeObj runtime.Object, reflectionType string) (*corev1.Pod, error) {
	var isNewObject bool

	if homeObj == nil {
		isNewObject = true

		homeObj = &corev1.Pod{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec:       corev1.PodSpec{},
		}
	}

	foreignPod := foreignObj.(*corev1.Pod)
	homePod := homeObj.(*corev1.Pod)

	foreignNamespace, err := f.nattingTable.DeNatNamespace(foreignPod.Namespace)
	if err != nil {
		return nil, err
	}

	f.forgeHomeMeta(&foreignPod.ObjectMeta, &homePod.ObjectMeta, foreignNamespace, reflectionType)
	delete(homePod.Labels, virtualKubelet.ReflectedpodKey)

	if isNewObject {
		homePod.Spec = f.forgePodSpec(foreignPod.Spec)
	}

	return homePod, nil
}

func (f *apiForger) podStatusForeignToHome(foreignObj, homeObj runtime.Object) *corev1.Pod {
	homePod := homeObj.(*corev1.Pod)
	foreignPod := foreignObj.(*corev1.Pod)

	homePod.Status = foreignPod.Status
	if homePod.Status.PodIP != "" {
		newIp, err := ChangePodIp(f.remoteRemappedPodCidr.Value().ToString(), foreignPod.Status.PodIP)
		if err != nil {
			klog.Error(err)
		}
		homePod.Status.PodIP = newIp
		homePod.Status.PodIPs[0].IP = newIp
	}

	if foreignPod.DeletionTimestamp != nil {
		homePod.DeletionTimestamp = nil
		foreignKey := fmt.Sprintf("%s/%s", foreignPod.Namespace, foreignPod.Name)
		reflectors.Blacklist[apimgmt.Pods][foreignKey] = struct{}{}
		klog.V(3).Infof("pod %s blacklisted because marked for deletion", foreignKey)
	}

	return homePod
}

// setPodToBeDeleted set the pod status such that it can be collected by the replicasetController,
// setting the pod status to PodUnknwon and all the containers in terminated status.
func (f *apiForger) setPodToBeDeleted(pod *corev1.Pod) *corev1.Pod {
	now := metav1.Now()

	pod.Status.Phase = corev1.PodUnknown
	for i := range pod.Status.ContainerStatuses {
		pod.Status.ContainerStatuses[i].State = corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{},
		}
	}
	pod.DeletionTimestamp = &now

	return pod
}

func (f *apiForger) podHomeToForeign(homeObj, foreignObj runtime.Object, reflectionType string) (*corev1.Pod, error) {
	var isNewObject bool
	var homePod, foreignPod *corev1.Pod

	if foreignObj == nil {
		isNewObject = true

		foreignPod = &corev1.Pod{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec:       corev1.PodSpec{},
		}
	} else {
		foreignPod = foreignObj.(*corev1.Pod)
	}

	homePod = homeObj.(*corev1.Pod)

	foreignNamespace, err := f.nattingTable.NatNamespace(homePod.Namespace, true)
	if err != nil {
		return nil, err
	}

	f.forgeForeignMeta(&homePod.ObjectMeta, &foreignPod.ObjectMeta, foreignNamespace, reflectionType)

	if isNewObject {
		foreignPod.Spec = f.forgePodSpec(homePod.Spec)
		foreignPod.Spec.Affinity = forgeAffinity()
	}

	return foreignPod, nil
}

func (f *apiForger) forgePodSpec(inputPodSpec corev1.PodSpec) corev1.PodSpec {
	outputPodSpec := corev1.PodSpec{}

	outputPodSpec.Volumes = forgeVolumes(inputPodSpec.Volumes)
	outputPodSpec.InitContainers = forgeContainers(inputPodSpec.InitContainers, outputPodSpec.Volumes)
	outputPodSpec.Containers = forgeContainers(inputPodSpec.Containers, outputPodSpec.Volumes)

	return outputPodSpec
}

func forgeContainers(inputContainers []corev1.Container, inputVolumes []corev1.Volume) []corev1.Container {
	containers := make([]corev1.Container, 0)

	for _, container := range inputContainers {
		volumeMounts := filterVolumeMounts(inputVolumes, container.VolumeMounts)
		containers = append(containers, translateContainer(container, volumeMounts))
	}

	return containers
}

func translateContainer(container corev1.Container, volumes []corev1.VolumeMount) corev1.Container {
	return corev1.Container{
		Name:            container.Name,
		Image:           container.Image,
		Command:         container.Command,
		Args:            container.Args,
		WorkingDir:      container.WorkingDir,
		Ports:           container.Ports,
		Env:             container.Env,
		Resources:       container.Resources,
		LivenessProbe:   container.LivenessProbe,
		ReadinessProbe:  container.ReadinessProbe,
		StartupProbe:    container.StartupProbe,
		SecurityContext: container.SecurityContext,
		VolumeMounts:    volumes,
	}
}

func forgeVolumes(volumesIn []corev1.Volume) []corev1.Volume {
	volumesOut := make([]corev1.Volume, 0)
	for _, v := range volumesIn {
		if v.ConfigMap != nil || v.EmptyDir != nil || v.DownwardAPI != nil {
			volumesOut = append(volumesOut, v)
		}
		// copy all volumes of type Secret except for the default token
		if v.Secret != nil && !strings.Contains(v.Secret.SecretName, "default-token") {
			volumesOut = append(volumesOut, v)
		}
	}
	return volumesOut
}

// remove from volumeMountsIn all the volumeMounts with name not contained in volumes
func filterVolumeMounts(volumes []corev1.Volume, volumeMountsIn []corev1.VolumeMount) []corev1.VolumeMount {
	volumeMounts := make([]corev1.VolumeMount, 0)
	for _, vm := range volumeMountsIn {
		for _, v := range volumes {
			if vm.Name == v.Name {
				volumeMounts = append(volumeMounts, vm)
			}
		}
	}
	return volumeMounts
}

// ChangePodIp creates a new IP address obtained by means of the old IP address and the new podCIDR
func ChangePodIp(newPodCidr string, oldPodIp string) (newPodIp string, err error) {
	if newPodCidr == tunnelEndpointCreator.DefaultPodCIDRValue {
		return oldPodIp, nil
	}
	// Parse newPodCidr
	ip, network, err := net.ParseCIDR(newPodCidr)
	if err != nil {
		return "", err
	}
	// Get mask
	mask := network.Mask
	// Get slice of bytes for newPodCidr
	// Type net.IP has underlying type []byte
	newIP := ip.To4()
	// Get oldPodIp as slice of bytes
	oldIP := net.ParseIP(oldPodIp)
	if oldIP == nil {
		return "", fmt.Errorf("cannot parse oldIp")
	}
	oldIP = oldIP.To4()
	// Substitute the last 32-mask bits of newPodCidr(newIP) with bits taken by the old ip
	for i := 0; i < len(mask); i++ {
		// Step 1: NOT(mask[i]) = mask[i] ^ 0xff. They are the 'host' bits
		// Step 2: BITWISE AND between the host bits and oldIP[i] zeroes the network bits in oldIP[i]
		// Step 3: BITWISE OR copies the result of step 2 in newIP[i]
		newIP[i] |= (mask[i] ^ 0xff) & oldIP[i]
	}
	return net.IP(newIP).String(), nil
}

func forgeAffinity() *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      liqoconst.TypeLabel,
								Operator: corev1.NodeSelectorOpNotIn,
								Values:   []string{affinitySelector},
							},
						},
					},
				},
			},
		},
	}
}
