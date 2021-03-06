package authservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"

	"github.com/liqotech/liqo/pkg/discovery"
)

func isNoContent(err error) bool {
	errStatus := &kerrors.StatusError{}
	if errors.As(err, &errStatus) {
		return errStatus.ErrStatus.Code == http.StatusNoContent
	}
	return false
}

func (authService *Controller) getServiceAccountCompleted(
	remoteClusterID string) (sa *v1.ServiceAccount, err error) {
	err = retry.OnError(
		retry.DefaultBackoff,
		func(err error) bool {
			err2 := authService.saInformer.GetStore().Resync()
			if err2 != nil {
				klog.Error(err)
				return false
			}
			return kerrors.IsNotFound(err)
		},
		func() error {
			sa, err = authService.getServiceAccount(remoteClusterID)
			return err
		},
	)

	err = retry.OnError(retry.DefaultBackoff, isNoContent, func() error {
		sa, err = authService.getServiceAccount(remoteClusterID)
		if err != nil {
			return err
		}

		if len(sa.Secrets) == 0 {
			return &kerrors.StatusError{ErrStatus: metav1.Status{
				Status: metav1.StatusFailure,
				Code:   http.StatusNoContent,
				Reason: metav1.StatusReasonNotFound,
			}}
		}

		return nil
	})
	return sa, err
}

func (authService *Controller) getServiceAccount(remoteClusterID string) (*v1.ServiceAccount, error) {
	tmp, exists, err := authService.saInformer.GetStore().GetByKey(
		strings.Join([]string{authService.namespace, fmt.Sprintf("remote-%s", remoteClusterID)}, "/"))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, kerrors.NewNotFound(schema.GroupResource{
			Resource: "serviceaccounts",
		}, remoteClusterID)
	}
	sa, ok := tmp.(*v1.ServiceAccount)
	if !ok {
		return nil, kerrors.NewNotFound(schema.GroupResource{
			Resource: "serviceaccounts",
		}, remoteClusterID)
	}
	return sa, nil
}

func (authService *Controller) createServiceAccount(remoteClusterID string) (*v1.ServiceAccount, error) {
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("remote-%s", remoteClusterID),
			Labels: map[string]string{
				discovery.LiqoManagedLabel: "true",
				discovery.ClusterIDLabel:   remoteClusterID,
			},
			// used to do garbage collection on cluster scoped resources (i.e. ClusterRole and ClusterRoleBinding)
			Finalizers: []string{
				discovery.GarbageCollection,
			},
		},
	}
	return authService.clientset.CoreV1().ServiceAccounts(
		authService.namespace).Create(context.TODO(), sa, metav1.CreateOptions{})
}
