package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var clientset *kubernetes.Clientset

// InitClient initializes the Kubernetes client
// Tries in-cluster config first, then falls back to kubeconfig
func InitClient() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, _ := os.UserHomeDir()
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes config: %w", err)
		}
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	return nil
}

// GetClientset returns the initialized Kubernetes clientset
func GetClientset() *kubernetes.Clientset {
	return clientset
}

// IsInitialized checks if the Kubernetes client is initialized
func IsInitialized() bool {
	return clientset != nil
}
