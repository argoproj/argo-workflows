# Basic Workflow Example

This example demonstrates how to submit a simple workflow using the Kubernetes client.

## What it does

- Loads kubeconfig from default location or specified path
- Creates a simple "Hello World" workflow
- Submits it to the cluster
- Prints the workflow details

## Running the example

```bash
# Use default kubeconfig location (~/.kube/config)
go run main.go

# Specify custom kubeconfig
go run main.go -kubeconfig /path/to/kubeconfig

# Specify namespace
go run main.go -namespace test
```

## Building

```bash
go build -o basic-workflow
./basic-workflow
```

## Expected output

```
Submitting workflow to namespace 'default'...
✓ Workflow submitted successfully!
  Name: hello-world-abc123
  Namespace: default
  UID: 12345678-1234-1234-1234-123456789012

View workflow status with:
  kubectl get workflow hello-world-abc123 -n default
  argo get hello-world-abc123 -n default
```

## Code walkthrough

1. **Parse flags**: Get kubeconfig path and namespace from command line
2. **Load config**: Use `clientcmd.BuildConfigFromFlags()` to load kubeconfig
3. **Create client**: Create workflow client for the namespace
4. **Define workflow**: Create a `Workflow` struct with a simple container
5. **Submit**: Call `Create()` with context to submit the workflow

## Next steps

- See `watch-workflow` example to track workflow progress
- See `grpc-client` example for remote Argo Server access
- See `workflow-template` example for reusable templates
- See `alternate_auth` example for production authentication patterns (in-cluster, token-based)
