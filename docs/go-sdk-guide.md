# Argo Workflows Go SDK Guide

The Argo Workflows Go SDK allows you to interact with Argo Workflows programmatically from Go applications. This guide covers installation, authentication, and common usage patterns.

## Installation

Add the Argo Workflows SDK to your Go project:

```bash
go get github.com/argoproj/argo-workflows/v4@latest
```

### Minimum Requirements

- Go 1.24 or later
- Kubernetes 1.28+ (if using Kubernetes client)
- Argo Workflows 3.4+ installed in your cluster

## Quick Start

Here's a simple example that submits a workflow:

```go
<!-- <embed id="quickstart" inject_from="code"> -->
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

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
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

<!-- </embed> -->
```

## Client Architecture

The Argo Workflows Go SDK provides two different client approaches for different use cases:

### 1. Kubernetes Client (Direct CRD Access)

**Use when:**

- You have kubeconfig access
- Running inside a Kubernetes cluster
- You want native Kubernetes API patterns
- You need watch/list operations with field selectors

**Package:** `github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned`

```go
import (
    wfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
    "k8s.io/client-go/tools/clientcmd"
)

// From kubeconfig
config, _ := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
clientset := wfclientset.NewForConfigOrDie(config)
wfClient := clientset.ArgoprojV1alpha1().Workflows(namespace)

// From in-cluster config
config, _ := rest.InClusterConfig()
clientset := wfclientset.NewForConfigOrDie(config)
```

### 2. Argo Server Client (gRPC/HTTP)

**Use when:**

- Accessing Argo Server remotely
- You don't have direct Kubernetes access
- You need service-oriented operations (retry, stop, suspend)
- Working with archived workflows

**Package:** `github.com/argoproj/argo-workflows/v4/pkg/apiclient`

```go
import (
    "github.com/argoproj/argo-workflows/v4/pkg/apiclient"
)

ctx, client, err := apiclient.NewClientFromOptsWithContext(ctx, apiclient.Opts{
    ArgoServerOpts: apiclient.ArgoServerOpts{
        URL: "localhost:2746",
    },
    AuthSupplier: func() string { return bearerToken },
})
if err != nil {
    panic(err)
}

serviceClient := client.NewWorkflowServiceClient(ctx)
```

### Comparison

| Feature | Kubernetes Client | Argo Server Client |
|---------|-------------------|-------------------|
| **Access Method** | Direct K8s API | gRPC/HTTP |
| **Authentication** | Kubeconfig/ServiceAccount | Bearer token/SSO |
| **Network** | Cluster access required | Remote access supported |
| **Operations** | CRUD, Watch, Patch | CRUD + Retry/Stop/Suspend |
| **Archived Workflows** | No | Yes |
| **Field Selectors** | Yes | Limited |
| **In-Cluster** | Optimal | Possible |

## Authentication

### Kubernetes Client Authentication

#### Using kubeconfig

```go
import (
    "k8s.io/client-go/tools/clientcmd"
    wfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
)

// Default kubeconfig location
config, err := clientcmd.BuildConfigFromFlags("",
    filepath.Join(os.Getenv("HOME"), ".kube", "config"))

// Custom kubeconfig location
config, err := clientcmd.BuildConfigFromFlags("", "/custom/path/to/kubeconfig")

// Create clientset
clientset := wfclientset.NewForConfig(config)
```

#### Using In-Cluster Config (for Pods)

```go
import (
    "k8s.io/client-go/rest"
    wfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
)

config, err := rest.InClusterConfig()
if err != nil {
    panic(err)
}

clientset := wfclientset.NewForConfig(config)
```

#### Using Service Account

When running in a pod, ensure your ServiceAccount has appropriate RBAC permissions:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: workflow-client
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: workflow-client-role
  namespace: default
rules:
- apiGroups: ["argoproj.io"]
  resources: ["workflows", "workflowtemplates", "cronworkflows"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: workflow-client-binding
  namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: workflow-client-role
subjects:
- kind: ServiceAccount
  name: workflow-client
  namespace: default
```

### Argo Server Client Authentication

#### Using Bearer Token

```go
import (
    "os"
    "github.com/argoproj/argo-workflows/v4/pkg/apiclient"
)

ctx, client, err := apiclient.NewClientFromOptsWithContext(ctx, apiclient.Opts{
    ArgoServerOpts: apiclient.ArgoServerOpts{
        URL:    "localhost:2746",
        Secure: true, // Use TLS
    },
    AuthSupplier: func() string {
        return os.Getenv("ARGO_TOKEN")
    },
})
```

#### Using kubeconfig (Argo Server in Kubernetes mode)

```go
import (
    "k8s.io/client-go/tools/clientcmd"
    "github.com/argoproj/argo-workflows/v4/pkg/apiclient"
)

ctx, client, err := apiclient.NewClientFromOptsWithContext(ctx, apiclient.Opts{
    ArgoServerOpts: apiclient.ArgoServerOpts{
        URL: "localhost:2746",
    },
    ClientConfigSupplier: func() clientcmd.ClientConfig {
        loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
        return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
            loadingRules,
            &clientcmd.ConfigOverrides{},
        )
    },
})
```

## Common Operations

### Creating Workflows

#### From a `struct`

```go
workflow := &wfv1.Workflow{
    ObjectMeta: metav1.ObjectMeta{
        GenerateName: "my-workflow-",
        Labels: map[string]string{
            "app": "my-app",
        },
    },
    Spec: wfv1.WorkflowSpec{
        Entrypoint: "main",
        Templates: []wfv1.Template{
            {
                Name: "main",
                Container: &corev1.Container{
                    Image:   "alpine:latest",
                    Command: []string{"sh", "-c"},
                    Args:    []string{"echo hello"},
                },
            },
        },
    },
}

created, err := wfClient.Create(ctx, workflow, metav1.CreateOptions{})
```

#### From YAML

```go
import (
    "os"
    "sigs.k8s.io/yaml"
)

// Read YAML file
data, err := os.ReadFile("workflow.yaml")
if err != nil {
    panic(err)
}

// Unmarshal into Workflow
var workflow wfv1.Workflow
if err := yaml.Unmarshal(data, &workflow); err != nil {
    panic(err)
}

// Submit
created, err := wfClient.Create(ctx, &workflow, metav1.CreateOptions{})
```

### Listing Workflows

```go
// List all workflows in namespace
list, err := wfClient.List(ctx, metav1.ListOptions{})
if err != nil {
    panic(err)
}

for _, wf := range list.Items {
    fmt.Printf("Workflow: %s, Phase: %s\n", wf.Name, wf.Status.Phase)
}

// List with label selector
list, err = wfClient.List(ctx, metav1.ListOptions{
    LabelSelector: "app=my-app",
})

// List with field selector
list, err = wfClient.List(ctx, metav1.ListOptions{
    FieldSelector: "status.phase=Running",
})
```

### Getting Workflow Details

```go
wf, err := wfClient.Get(ctx, "workflow-name", metav1.GetOptions{})
if err != nil {
    panic(err)
}

fmt.Printf("Name: %s\n", wf.Name)
fmt.Printf("Phase: %s\n", wf.Status.Phase)
fmt.Printf("Started: %s\n", wf.Status.StartedAt)
fmt.Printf("Finished: %s\n", wf.Status.FinishedAt)

// Access node statuses
for nodeName, nodeStatus := range wf.Status.Nodes {
    fmt.Printf("Node %s: %s\n", nodeName, nodeStatus.Phase)
}
```

### Watching Workflows

```go
<!-- <embed id="watch-workflow" inject_from="code"> -->
func watchWorkflow(ctx context.Context, wfClient v1alpha1.WorkflowInterface, name string) error {
	// Create field selector to watch only this workflow
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	// Start watching with a timeout
	watchIf, err := wfClient.Watch(ctx, metav1.ListOptions{
		FieldSelector:   fieldSelector.String(),
		TimeoutSeconds:  ptr.To(int64(300)), // 5 minute timeout
		ResourceVersion: "0",
	})
	if err != nil {
		return fmt.Errorf("failed to watch workflow: %w", err)
	}
	defer watchIf.Stop()

	// Track last seen phase to avoid duplicate messages
	lastPhase := wfv1.WorkflowUnknown
	startTime := time.Now()

	// Process watch events
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-watchIf.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed unexpectedly")
			}

			wf, ok := event.Object.(*wfv1.Workflow)
			if !ok {
				continue
			}

			switch event.Type {
			case watch.Added:
				fmt.Printf("[%s] Workflow created\n", formatDuration(time.Since(startTime)))

			case watch.Modified:
				// Only print if phase changed
				if wf.Status.Phase != lastPhase {
					lastPhase = wf.Status.Phase
					fmt.Printf("[%s] Phase: %s\n", formatDuration(time.Since(startTime)), wf.Status.Phase)

					// Print additional details based on phase
					if wf.Status.Phase == wfv1.WorkflowRunning && !wf.Status.StartedAt.IsZero() {
						fmt.Printf("         Started at: %s\n", wf.Status.StartedAt.Format(time.RFC3339))
					}
				}

				// Check if workflow is complete
				if !wf.Status.FinishedAt.IsZero() {
					fmt.Println("─────────────────────────────────────────────")
					fmt.Printf("✓ Workflow completed!\n")
					fmt.Printf("  Final Phase: %s\n", wf.Status.Phase)
					fmt.Printf("  Started: %s\n", wf.Status.StartedAt.Format(time.RFC3339))
					fmt.Printf("  Finished: %s\n", wf.Status.FinishedAt.Format(time.RFC3339))
					fmt.Printf("  Duration: %s\n", wf.Status.FinishedAt.Sub(wf.Status.StartedAt.Time))

					if wf.Status.Message != "" {
						fmt.Printf("  Message: %s\n", wf.Status.Message)
					}

					// Print node statuses
					if len(wf.Status.Nodes) > 0 {
						fmt.Printf("\nNode Details:\n")
						for nodeName, node := range wf.Status.Nodes {
							fmt.Printf("  - %s: %s\n", nodeName, node.Phase)
						}
					}

					return nil
				}

			case watch.Deleted:
				fmt.Printf("[%s] Workflow deleted\n", formatDuration(time.Since(startTime)))
				return nil
			}
		}
	}
}
<!-- </embed> -->
```

### Deleting Workflows

```go
// Delete single workflow
err := wfClient.Delete(ctx, "workflow-name", metav1.DeleteOptions{})

// Delete with propagation policy
err = wfClient.Delete(ctx, "workflow-name", metav1.DeleteOptions{
    PropagationPolicy: &deletePropagationBackground,
})

// Delete collection (multiple workflows)
err = wfClient.DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
    LabelSelector: "app=my-app",
})
```

### Using Argo Server Client

```go
<!-- <embed id="grpc-client-operations" inject_from="code"> -->
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
<!-- </embed> -->
```

## Working with Workflow Templates

### Creating WorkflowTemplates

```go
<!-- <embed id="create-workflow-template" inject_from="code"> -->
	template := &wfv1.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello-world",
			Labels: map[string]string{
				"example": "workflow-template",
			},
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "greet",
			Arguments: wfv1.Arguments{
				Parameters: []wfv1.Parameter{
					{
						Name:  "message",
						Value: wfv1.AnyStringPtr("Hello World"),
					},
				},
			},
			Templates: []wfv1.Template{
				{
					Name: "greet",
					Inputs: wfv1.Inputs{
						Parameters: []wfv1.Parameter{
							{Name: "message"},
						},
					},
					Container: &corev1.Container{
						Image:   "busybox:latest",
						Command: []string{"echo"},
						Args:    []string{"{{inputs.parameters.message}}"},
					},
				},
			},
		},
	}

	var createdTemplate *wfv1.WorkflowTemplate
	existingTemplate, err := wftClient.Get(ctx, template.Name, metav1.GetOptions{})
	if err == nil {
		template.ResourceVersion = existingTemplate.ResourceVersion
		createdTemplate, err = wftClient.Update(ctx, template, metav1.UpdateOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating template: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ WorkflowTemplate '%s' updated (already existed)\n\n", createdTemplate.Name)
	} else {
		createdTemplate, err = wftClient.Create(ctx, template, metav1.CreateOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating template: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ WorkflowTemplate '%s' created\n\n", createdTemplate.Name)
	}
<!-- </embed> -->
```

### Submitting from WorkflowTemplate

```go
<!-- <embed id="submit-from-template" inject_from="code"> -->
	workflow1 := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "from-template-default-",
		},
		Spec: wfv1.WorkflowSpec{
			WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{
				Name: createdTemplate.Name,
			},
		},
	}

	submitted1, err := wfClient.Create(ctx, workflow1, metav1.CreateOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error submitting workflow: %v\n", err)
		cleanup(ctx, wftClient, createdTemplate.Name)
		os.Exit(1)
	}
	fmt.Printf("✓ Workflow '%s' submitted with default parameters\n\n", submitted1.Name)
<!-- </embed> -->
```

### Working with CronWorkflows

```go
cronClient := clientset.ArgoprojV1alpha1().CronWorkflows(namespace)

cronWf := &wfv1.CronWorkflow{
    ObjectMeta: metav1.ObjectMeta{
        Name: "my-cron-workflow",
    },
    Spec: wfv1.CronWorkflowSpec{
        Schedule: "*/5 * * * *", // Every 5 minutes
        WorkflowSpec: wfv1.WorkflowSpec{
            WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{
                Name: "my-template",
            },
        },
    },
}

created, err := cronClient.Create(ctx, cronWf, metav1.CreateOptions{})
```

## Advanced Topics

### Using Informers for Event-Driven Applications

Informers provide efficient caching and watching of resources:

```go
import (
    "k8s.io/client-go/tools/cache"
    wfinformers "github.com/argoproj/argo-workflows/v4/pkg/client/informers/externalversions"
)

// Create informer factory
informerFactory := wfinformers.NewSharedInformerFactory(clientset, 0)
wfInformer := informerFactory.Argoproj().V1alpha1().Workflows()

// Add event handlers
wfInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) {
        wf := obj.(*wfv1.Workflow)
        fmt.Printf("Workflow added: %s\n", wf.Name)
    },
    UpdateFunc: func(oldObj, newObj interface{}) {
        wf := newObj.(*wfv1.Workflow)
        fmt.Printf("Workflow updated: %s, phase: %s\n", wf.Name, wf.Status.Phase)
    },
    DeleteFunc: func(obj interface{}) {
        wf := obj.(*wfv1.Workflow)
        fmt.Printf("Workflow deleted: %s\n", wf.Name)
    },
})

// Start informer
stopCh := make(chan struct{})
defer close(stopCh)
informerFactory.Start(stopCh)
informerFactory.WaitForCacheSync(stopCh)

// Keep running
<-stopCh
```

### Using Listers for Efficient Querying

```go
import (
    wflisters "github.com/argoproj/argo-workflows/v4/pkg/client/listers/workflow/v1alpha1"
)

// Create lister from informer
lister := wfInformer.Lister()

// List workflows (from cache)
workflows, err := lister.Workflows(namespace).List(labels.Everything())

// Get specific workflow (from cache)
wf, err := lister.Workflows(namespace).Get("workflow-name")
```

### Testing with Fake Clients

```go
import (
    fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
)

// Create fake clientset for testing
fakeClient := fakewfclientset.NewSimpleClientset()
wfClient := fakeClient.ArgoprojV1alpha1().Workflows(namespace)

// Use as normal
created, err := wfClient.Create(ctx, &workflow, metav1.CreateOptions{})
```

## Best Practices

### 1. Use Context

Pass context through your application for cancellation and timeout control:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

wf, err := wfClient.Create(ctx, &workflow, metav1.CreateOptions{})
```

### 2. Handle Errors Appropriately

```go
import (
    apierrors "k8s.io/apimachinery/pkg/api/errors"
)

wf, err := wfClient.Get(ctx, name, metav1.GetOptions{})
if err != nil {
    if apierrors.IsNotFound(err) {
        // Workflow doesn't exist
        fmt.Printf("Workflow %s not found\n", name)
    } else {
        // Other error
        return fmt.Errorf("failed to get workflow: %w", err)
    }
}
```

### 3. Use `GenerateName` for Unique Workflows

```go
workflow := &wfv1.Workflow{
    ObjectMeta: metav1.ObjectMeta{
        GenerateName: "my-workflow-", // Will generate my-workflow-xyz123
    },
    // ...
}
```

## Additional Resources

- [API Reference](https://pkg.go.dev/github.com/argoproj/argo-workflows/v4)
- [Workflow Examples](https://github.com/argoproj/argo-workflows/tree/main/examples/) - YAML examples of workflows
- [Argo Workflows Documentation](https://argo-workflows.readthedocs.io/)

## Getting Help

- [Slack Channel](https://argoproj.github.io/community/join-slack)
- [GitHub Issues](https://github.com/argoproj/argo-workflows/issues)
- [GitHub Discussions](https://github.com/argoproj/argo-workflows/discussions)
