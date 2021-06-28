/*
Copyright 2021.

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

package resourceoffercontroller

import (
	"context"
	"sync"
	"time"

	v1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	configv1alpha1 "github.com/liqotech/liqo/apis/config/v1alpha1"
	sharingv1alpha1 "github.com/liqotech/liqo/apis/sharing/v1alpha1"
	crdreplicator "github.com/liqotech/liqo/internal/crdReplicator"
	"github.com/liqotech/liqo/pkg/clusterid"
)

// ResourceOfferReconciler reconciles a ResourceOffer object.
type ResourceOfferReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	eventsRecorder record.EventRecorder
	clusterID      clusterid.ClusterID

	virtualKubeletImage     string
	initVirtualKubeletImage string

	resyncPeriod       time.Duration
	configuration      *configv1alpha1.ClusterConfig
	configurationMutex sync.RWMutex
}

//+kubebuilder:rbac:groups=sharing.liqo.io,resources=resourceoffers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sharing.liqo.io,resources=resourceoffers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sharing.liqo.io,resources=resourceoffers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ResourceOfferReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	klog.V(4).Infof("reconciling ResourceOffer %v", req.NamespacedName)
	// get resource offer
	var resourceOffer sharingv1alpha1.ResourceOffer
	if err = r.Get(ctx, req.NamespacedName, &resourceOffer); err != nil {
		if kerrors.IsNotFound(err) {
			// reconcile was triggered by a delete request
			klog.Infof("ResourceRequest %v deleted", req.NamespacedName)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		// not managed error
		klog.Error(err)
		return ctrl.Result{}, err
	}

	// we do that on ResourceOffer creation
	if metav1.GetControllerOf(&resourceOffer) == nil {
		if err = r.setControllerReference(ctx, &resourceOffer); err != nil {
			klog.Error(err)
			return ctrl.Result{}, err
		}
		if err = r.Client.Update(ctx, &resourceOffer); err != nil {
			klog.Error(err)
			return ctrl.Result{}, err
		}
		// we always return after a metadata or spec update to have a clean resource where to work
		return ctrl.Result{}, nil
	}

	result = ctrl.Result{RequeueAfter: r.resyncPeriod}

	// defer the status update function
	defer func() {
		if newErr := r.Client.Status().Update(ctx, &resourceOffer); newErr != nil {
			klog.Error(newErr)
			err = newErr
		}
	}()

	// filter resource offers and create a virtual-kubelet only for the good ones
	if err = r.setResourceOfferPhase(ctx, &resourceOffer); err != nil {
		klog.Error(err)
		return ctrl.Result{}, err
	}

	// check the virtual kubelet deployment
	if err = r.checkVirtualKubeletDeployment(ctx, &resourceOffer); err != nil {
		klog.Error(err)
		return ctrl.Result{}, err
	}

	// delete the ClusterRoleBinding if the VirtualKubelet Deployment is not up
	if resourceOffer.Status.VirtualKubeletStatus == sharingv1alpha1.VirtualKubeletStatusNone {
		if err = r.deleteClusterRoleBinding(ctx, &resourceOffer); err != nil {
			klog.Error(err)
			return ctrl.Result{}, err
		}
	}

	if !isAccepted(&resourceOffer) || !resourceOffer.DeletionTimestamp.IsZero() {
		// delete virtual kubelet deployment
		if err = r.deleteVirtualKubeletDeployment(ctx, &resourceOffer); err != nil {
			klog.Error(err)
			return ctrl.Result{}, err
		}
		return result, nil
	}

	// create the virtual kubelet deployment
	if err = r.createVirtualKubeletDeployment(ctx, &resourceOffer); err != nil {
		klog.Error(err)
		return ctrl.Result{}, err
	}

	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceOfferReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p, err := predicate.LabelSelectorPredicate(crdreplicator.ReplicatedResourcesLabelSelector)
	if err != nil {
		klog.Error(err)
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&sharingv1alpha1.ResourceOffer{}, builder.WithPredicates(p)).
		Owns(&v1.Deployment{}).
		Complete(r)
}
