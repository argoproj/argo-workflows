// <snip id="quickstart">
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
)

func main() {
	// Parse command-line flags
	var (
		kubeconfig = flag.String("kubeconfig", defaultKubeconfig(), "path to kubeconfig file")
		namespace  = flag.String("namespace", "argo", "namespace to submit workflow")
	)
	flag.Parse()

	ctx := context.Background()

	// Load kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Create workflow client
	wfClient := wfclientset.NewForConfigOrDie(config).
		ArgoprojV1alpha1().
		Workflows(*namespace)

	// Define a simple workflow
	workflow := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "hello-world-",
			Labels: map[string]string{
				"example": "basic-workflow",
			},
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "hello-world",
			Templates: []wfv1.Template{
				{
					Name: "hello-world",
					Container: &corev1.Container{
						Image:   "busybox:latest",
						Command: []string{"echo"},
						Args:    []string{"Hello, World from Argo Workflows Go SDK!"},
					},
				},
			},
		},
	}

	// Submit the workflow
	fmt.Printf("Submitting workflow to namespace '%s'...\n", *namespace)
	created, err := wfClient.Create(ctx, workflow, metav1.CreateOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating workflow: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Workflow submitted successfully!\n")
	fmt.Printf("  Name: %s\n", created.Name)
	fmt.Printf("  Namespace: %s\n", created.Namespace)
	fmt.Printf("  UID: %s\n", created.UID)
	fmt.Printf("\nView workflow status with:\n")
	fmt.Printf("  kubectl get workflow %s -n %s\n", created.Name, created.Namespace)
	fmt.Printf("  argo get %s -n %s\n", created.Name, created.Namespace)
}

// defaultKubeconfig returns the default kubeconfig path
func defaultKubeconfig() string {
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}

// </snip>
