# Argo Workflows Go SDK Examples

This directory contains complete, working examples demonstrating how to use the Argo Workflows Go SDK.
This is using the code in argo-workflows codebase as an SDK to build and control workflows.

## Prerequisites

- Go 1.24.10
- Access to a Kubernetes cluster with Argo Workflows installed, and able to run workflows in the `argo` namespace
- Kubeconfig configured for cluster access
- kubectl configured with cluster access (for Kubernetes client examples)
- Argo Server running (for gRPC client examples)

## Available Examples

### [basic-workflow/](./basic-workflow/)
Simple workflow submission using the Kubernetes client.

**Learn:** Loading kubeconfig, creating workflow client, submitting workflows

```bash
cd basic-workflow
go run main.go
```

### [watch-workflow/](./watch-workflow/)
Submit a workflow and watch its progress in real-time.

**Learn:** Watching workflows, handling events, field selectors, context handling

```bash
cd watch-workflow
go run main.go
```

### [grpc-client/](./grpc-client/)
Interact with Argo Server using the gRPC client.

**Learn:** Argo Server connection, gRPC client, service methods, remote access

```bash
cd grpc-client
export ARGO_SERVER=localhost:2746
export ARGO_TOKEN=<your-token>
go run main.go
```

### [workflow-template/](./workflow-template/)
Create and use WorkflowTemplates for reusable workflows.

**Learn:** WorkflowTemplates, parameters, template references, listing by labels

```bash
cd workflow-template
go run main.go
```

### [alternate_auth/](./alternate_auth/)
Demonstrates alternate authentication methods (in-cluster, token-based).

**Learn:** In-cluster authentication, bearer tokens, RBAC, production deployment patterns

```bash
cd alternate_auth
go run main.go -mode incluster
# or
go run main.go -mode token
```

**Note:** For basic kubeconfig authentication, see `basic-workflow/`. For gRPC authentication, see `grpc-client/`.

## Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/argoproj/argo-workflows.git
   cd argo-workflows/examples/go-sdk
   ```

2. **Choose an example**
   ```bash
   cd basic-workflow
   ```

3. **Run the example**
   ```bash
   go run main.go
   ```

## Building Examples

Each example can be built into a standalone binary:

```bash
cd basic-workflow
go build -o basic-workflow
./basic-workflow
```

## Example Structure

Each example includes:
- `main.go` - Complete, working Go code
- `go.mod` - Go module definition
- `README.md` - Detailed documentation and usage

## Two Client Approaches

The SDK provides two ways to interact with Argo Workflows:

### Kubernetes Client (Direct CRD Access)

Used in: `basic-workflow/`, `watch-workflow/`, `workflow-template/`, `alternate_auth/`

```go
import wfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"

config, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
wfClient := wfclientset.NewForConfigOrDie(config).
    ArgoprojV1alpha1().
    Workflows(namespace)
```

**Best for:**
- In-cluster applications
- kubectl-like tools
- Direct Kubernetes API access
- Efficient watching and listing

### Argo Server Client (gRPC/HTTP)

Used in: `grpc-client/`

```go
import "github.com/argoproj/argo-workflows/v4/pkg/apiclient"

ctx, client, _ := apiclient.NewClientFromOptsWithContext(ctx, apiclient.Opts{
    ArgoServerOpts: apiclient.ArgoServerOpts{URL: "localhost:2746"},
    AuthSupplier: func() string { return token },
})
serviceClient := client.NewWorkflowServiceClient(ctx)
```

**Best for:**
- Remote access
- External applications
- Using Argo Server features (archives, etc.)
- Service-oriented operations (retry, suspend, etc.)

## Common Patterns

### Submit a Workflow

```go
ctx := context.Background()
workflow := &wfv1.Workflow{
    ObjectMeta: metav1.ObjectMeta{
        GenerateName: "my-workflow-",
    },
    Spec: wfv1.WorkflowSpec{
        Entrypoint: "main",
        Templates: []wfv1.Template{
            {
                Name: "main",
                Container: &corev1.Container{
                    Image: "busybox",
                    Command: []string{"echo", "hello"},
                },
            },
        },
    },
}

created, err := wfClient.Create(ctx, workflow, metav1.CreateOptions{})
```

### List Workflows

```go
list, err := wfClient.List(ctx, metav1.ListOptions{})
for _, wf := range list.Items {
    fmt.Printf("%s: %s\n", wf.Name, wf.Status.Phase)
}
```

### Watch Workflow Progress

```go
fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))
watchIf, err := wfClient.Watch(ctx, metav1.ListOptions{
    FieldSelector: fieldSelector.String(),
})
defer watchIf.Stop()

for event := range watchIf.ResultChan() {
    wf := event.Object.(*wfv1.Workflow)
    fmt.Printf("Phase: %s\n", wf.Status.Phase)
}
```

## Testing Examples

All examples are designed to compile and run successfully. To test:

```bash
# Test compilation
cd examples/go-sdk
for dir in */; do
    echo "Testing $dir..."
    (cd "$dir" && go build)
done

# Run with dry-run/test mode
cd basic-workflow
go run main.go --help
```

## Environment Variables

Common environment variables used in examples:

- `KUBECONFIG` - Path to kubeconfig file
- `ARGO_SERVER` - Argo Server address (e.g., `localhost:2746`)
- `ARGO_TOKEN` - Bearer token for Argo Server authentication
- `ARGO_NAMESPACE` - Default namespace for workflows
- `KUBE_TOKEN` - Kubernetes bearer token

## Troubleshooting

### "Error loading kubeconfig"
- Verify kubeconfig path is correct
- Check file permissions
- Try with `-kubeconfig` flag

### "Error creating workflow: Unauthorized"
- Verify ServiceAccount has RBAC permissions
- Check token is valid
- Ensure namespace access

### "Connection refused"
- Verify Kubernetes cluster is accessible
- For gRPC examples: ensure Argo Server is running
- Check port forwarding: `kubectl -n argo port-forward svc/argo-server 2746:2746`

### "No such file or directory"
- Run examples from their respective directories
- Or use absolute paths for `-kubeconfig` flag

## Documentation

- [Go SDK Guide](../../docs/go-sdk-guide.md) - Comprehensive SDK documentation
- [Migration Guide](../../docs/go-sdk-migration-guide.md) - Migrating to v3.7+
- [API Reference](https://pkg.go.dev/github.com/argoproj/argo-workflows/v4)
- [Argo Workflows Docs](https://argo-workflows.readthedocs.io/)

## Getting Help

- [Slack Channel](https://argoproj.github.io/community/join-slack)
- [GitHub Issues](https://github.com/argoproj/argo-workflows/issues)
- [GitHub Discussions](https://github.com/argoproj/argo-workflows/discussions)

## Contributing

Found an issue or have an improvement? Please:
1. Open an issue describing the problem/enhancement
2. Submit a pull request with your changes
3. Ensure examples compile and run successfully

## Next Steps

After exploring these examples:

1. Read the [Go SDK Guide](../../docs/go-sdk-guide.md) for in-depth documentation
2. Check the [YAML examples](../) for workflow patterns
3. Review the [CLI source code](../../cmd/argo/commands/) for advanced usage
4. Join the community on [Slack](https://argoproj.github.io/community/join-slack)
