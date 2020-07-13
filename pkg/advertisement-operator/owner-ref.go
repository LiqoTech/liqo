package advertisement_operator

import (
	protocolv1 "github.com/liqoTech/liqo/api/advertisement-operator/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func GetOwnerReference(object interface{}) []metav1.OwnerReference {

	ownerRef := make([]metav1.OwnerReference, 1)

	switch obj := object.(type) {
	case *appsv1.Deployment:
		ownerRef = []metav1.OwnerReference{
			{
				APIVersion: obj.APIVersion,
				Kind:       obj.Kind,
				Name:       obj.Name,
				UID:        obj.UID,
			},
		}
	case *protocolv1.Advertisement:
		ownerRef = []metav1.OwnerReference{
			{
				APIVersion: obj.APIVersion,
				Kind:       obj.Kind,
				Name:       obj.Name,
				UID:        obj.UID,
			},
		}
	default:
		klog.Error("Invalid type for owner reference")
	}

	return ownerRef
}
