package kube

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetaListOpts, MetaGetOpts, MetaCreateOpts, MetaUpdateOpts, MetaDeleteOpts
// and IsNotFound/IsAlreadyExists are small client-go wrappers shared by
// every Kubernetes-backed adapter subpackage (kubestore, kubecrypt), so
// they live at this shared level rather than being duplicated per adapter.

func MetaListOpts(labelKey string) metav1.ListOptions {
	return metav1.ListOptions{LabelSelector: labelKey}
}

func MetaGetOpts() metav1.GetOptions {
	return metav1.GetOptions{}
}

func MetaCreateOpts() metav1.CreateOptions {
	return metav1.CreateOptions{}
}

func MetaUpdateOpts() metav1.UpdateOptions {
	return metav1.UpdateOptions{}
}

func MetaDeleteOpts() metav1.DeleteOptions {
	return metav1.DeleteOptions{}
}

func IsNotFound(err error) bool {
	return apierrors.IsNotFound(err)
}

func IsAlreadyExists(err error) bool {
	return apierrors.IsAlreadyExists(err)
}
