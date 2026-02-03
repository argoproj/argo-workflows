package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func main() {
	var (
		argoServer = flag.String("argo-server", getEnvOrDefault("ARGO_SERVER", "localhost:2746"), "Argo Server address")
		token      = flag.String("token", os.Getenv("ARGO_TOKEN"), "Bearer token for authentication")
		namespace  = flag.String("namespace", "argo", "namespace for workflow")
		secure     = flag.Bool("secure", true, "whether the Argo Server uses TLS")
		insecure   = flag.Bool("insecure-skip-verify", false, "skip TLS certificate verification")
	)
	flag.Parse()

	if *argoServer == "" {
		fmt.Fprintf(os.Stderr, "Error: --argo-server is required (or set ARGO_SERVER env var)\n")
		os.Exit(1)
	}

	ctx := context.Background()

	// Create Argo Server client
	fmt.Printf("Connecting to Argo Server at %s...\n", *argoServer)

	// <embed id="grpc-client-operations">
	ctx, client, err := apiclient.NewClientFromOptsWithContext(ctx, apiclient.Opts{
		ArgoServerOpts: apiclient.ArgoServerOpts{
			URL:                *argoServer,
			Secure:             *secure,
			InsecureSkipVerify: *insecure,
		},
		AuthSupplier: func() string {
			if *token != "" {
				return *token
			}
			return ""
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	// Create workflow service client
	serviceClient := client.NewWorkflowServiceClient(ctx)

	// Define a simple workflow
	workflow := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "grpc-example-",
			Labels: map[string]string{
				"example": "grpc-client",
			},
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "hello",
			Templates: []wfv1.Template{
				{
					Name: "hello",
					Container: &corev1.Container{
						Image:   "busybox:latest",
						Command: []string{"echo"},
						Args:    []string{"Hello from gRPC client!"},
					},
				},
			},
		},
	}

	// Submit the workflow
	fmt.Printf("Submitting workflow to namespace '%s'...\n", *namespace)
	created, err := serviceClient.CreateWorkflow(ctx, &workflowpkg.WorkflowCreateRequest{
		Namespace: *namespace,
		Workflow:  workflow,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating workflow: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Workflow submitted successfully!\n")
	fmt.Printf("  Name: %s\n", created.Name)
	fmt.Printf("  Namespace: %s\n", created.Namespace)
	fmt.Printf("  UID: %s\n", created.UID)

	// Get workflow details
	time.Sleep(time.Second)
	fmt.Printf("\nFetching workflow details...\n")
	wf, err := serviceClient.GetWorkflow(ctx, &workflowpkg.WorkflowGetRequest{
		Namespace: *namespace,
		Name:      created.Name,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting workflow: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Phase: %s\n", wf.Status.Phase)
	if !wf.Status.StartedAt.IsZero() {
		fmt.Printf("  Started: %s\n", wf.Status.StartedAt.Format("2006-01-02 15:04:05"))
	}

	// List workflows
	fmt.Printf("\nListing recent workflows in namespace '%s'...\n", *namespace)
	list, err := serviceClient.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{
		Namespace: *namespace,
		ListOptions: &metav1.ListOptions{
			Limit: 5,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing workflows: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d workflow(s):\n", len(list.Items))
	for i, wf := range list.Items {
		fmt.Printf("  %d. %s (%s)\n", i+1, wf.Name, wf.Status.Phase)
	}
	// </embed>

	fmt.Printf("\nView workflow with:\n")
	fmt.Printf("  argo get %s -n %s\n", created.Name, *namespace)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
