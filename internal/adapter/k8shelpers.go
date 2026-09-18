package adapter

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func metaListOpts(labelKey string) metav1.ListOptions {
	return metav1.ListOptions{LabelSelector: labelKey}
}

func metaGetOpts() metav1.GetOptions {
	return metav1.GetOptions{}
}

func metaCreateOpts() metav1.CreateOptions {
	return metav1.CreateOptions{}
}

func metaUpdateOpts() metav1.UpdateOptions {
	return metav1.UpdateOptions{}
}

func metaDeleteOpts() metav1.DeleteOptions {
	return metav1.DeleteOptions{}
}

func isNotFound(err error) bool {
	return apierrors.IsNotFound(err)
}

func isAlreadyExists(err error) bool {
	return apierrors.IsAlreadyExists(err)
}
