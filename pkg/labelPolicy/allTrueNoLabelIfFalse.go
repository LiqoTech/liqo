package labelPolicy

import corev1 "k8s.io/api/core/v1"

type AllTrueNoLabelIfFalse struct{}

func (at *AllTrueNoLabelIfFalse) Process(physicalNodes *corev1.NodeList, key string) (value string, insertLabel bool) {
	for _, node := range physicalNodes.Items {
		if v, ok := node.Labels[key]; !ok || (v != "true" && v != "") {
			return "", false
		}
	}
	return "", true
}
