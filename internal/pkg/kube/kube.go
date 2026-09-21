// Package kube holds generic Kubernetes client-go plumbing shared by every
// adapter subpackage under internal/adapter (kubestore, kubecrypt):
// building a clientset from in-cluster config or a kubeconfig file, and
// small metav1/apierrors wrappers. It knows nothing about IAM's domain —
// just Kubernetes.
package kube

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// BuildClientset returns a Kubernetes clientset, using in-cluster config
// when kubeconfigPath is empty and no KUBECONFIG-style file is reachable,
// falling back to the given kubeconfig file otherwise.
func BuildClientset(kubeconfigPath string) (kubernetes.Interface, error) {
	cfg, err := restConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("building kube client config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("building kube clientset: %w", err)
	}
	return clientset, nil
}

// BuildDynamicClient returns a Kubernetes dynamic client, for talking to
// custom resources (like ecp's Role/RoleAssignment CRDs, see
// internal/adapter/kuberbac) that have no generated typed clientset in
// this module. Same config resolution as BuildClientset.
func BuildDynamicClient(kubeconfigPath string) (dynamic.Interface, error) {
	cfg, err := restConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("building kube client config: %w", err)
	}
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("building kube dynamic client: %w", err)
	}
	return client, nil
}

func restConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath == "" {
		if cfg, err := rest.InClusterConfig(); err == nil {
			return cfg, nil
		}
		// Not running in-cluster and no explicit path given: fall back to
		// the standard kubeconfig discovery (KUBECONFIG env var, then
		// ~/.kube/config), matching kubectl's own default behavior.
		// clientcmd.BuildConfigFromFlags("", "") does NOT do this on its
		// own - it only tries in-cluster config for an empty path and
		// otherwise fails, since its ClientConfigLoadingRules has no
		// Precedence list set for the default paths.
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}).ClientConfig()
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}
