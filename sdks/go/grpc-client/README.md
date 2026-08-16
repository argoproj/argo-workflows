# gRPC Client Example

This example demonstrates how to interact with Argo Server using the gRPC client.

## See first

Look at `basic-workflow` for a simpler starting example.

## What it does

- Connects to Argo Server via gRPC
- Authenticates using a bearer token
- Submits a workflow using the workflow service client
- Retrieves workflow details
- Lists recent workflows in the namespace

## Running the example

### Setup Argo Server

First, ensure Argo Server is running:

```bash
# Port forward Argo Server (if running in cluster)
kubectl -n argo port-forward svc/argo-server 2746:2746

# Or if Argo Server is exposed externally, use that URL
```

### Get Authentication Token

```bash
# Get token from secret (if using token auth)
ARGO_TOKEN=$(kubectl -n argo get secret <secret-name> -o jsonpath='{.data.token}' | base64 -d)
export ARGO_TOKEN

# Or create a service account token
kubectl -n argo create token argo-server
```

### Run the example

```bash
# With environment variable
export ARGO_SERVER=localhost:2746
export ARGO_TOKEN=<your-token>
go run main.go

# With command-line flags
go run main.go -argo-server localhost:2746 -token <your-token>

# For insecure connection (development only)
go run main.go -argo-server localhost:2746 -secure=false

# Skip TLS verification (development only)
go run main.go -argo-server localhost:2746 -insecure-skip-verify
```

## Expected output

```
Connecting to Argo Server at localhost:2746...
Submitting workflow to namespace 'default'...
✓ Workflow submitted successfully!
  Name: grpc-example-abc123
  Namespace: default
  UID: 12345678-1234-1234-1234-123456789012

Fetching workflow details...
  Phase: Pending
  Started: 2025-01-15 10:30:00

Listing recent workflows in namespace 'default'...
Found 5 workflow(s):
  1. grpc-example-abc123 (Pending)
  2. previous-workflow-xyz789 (Succeeded)
  3. another-workflow-def456 (Running)
  4. old-workflow-ghi789 (Failed)
  5. test-workflow-jkl012 (Succeeded)

View workflow with:
  argo get grpc-example-abc123 -n default
```

## Code walkthrough

1. **Parse flags**: Get Argo Server URL, token, and TLS settings
2. **Create client**: Use `NewClientFromOptsWithContext()` to create Argo client
3. **Get service client**: Create workflow service client from main client
4. **Submit workflow**: Use `CreateWorkflow()` with workflow request
5. **Get details**: Use `GetWorkflow()` to fetch workflow status
6. **List workflows**: Use `ListWorkflows()` to see recent workflows

## Key concepts

### Argo Server Options

The `ArgoServerOpts` struct configures the connection:

```go
ArgoServerOpts: apiclient.ArgoServerOpts{
    URL:                "localhost:2746",  // Server address
    Secure:             true,               // Use TLS
    InsecureSkipVerify: false,             // Verify TLS certificates
    HTTP1:              false,             // Use gRPC (not HTTP/1)
}
```

### Authentication

The `AuthSupplier` function provides the bearer token:

```go
AuthSupplier: func() string {
    return os.Getenv("ARGO_TOKEN")
}
```

### Service Client Methods

The workflow service client provides many operations:

- `CreateWorkflow()` - Submit new workflow
- `GetWorkflow()` - Get workflow details
- `ListWorkflows()` - List workflows
- `DeleteWorkflow()` - Delete workflow
- `RetryWorkflow()` - Retry failed workflow
- `StopWorkflow()` - Stop running workflow
- `SuspendWorkflow()` - Suspend workflow
- `ResumeWorkflow()` - Resume suspended workflow
- `TerminateWorkflow()` - Terminate workflow

## Comparison with Kubernetes Client

| Feature | gRPC Client | Kubernetes Client |
|---------|-------------|-------------------|
| **Use case** | Remote access | In-cluster or kubectl access |
| **Protocol** | gRPC/HTTP | Kubernetes API |
| **Auth** | Bearer token/SSO | Kubeconfig/ServiceAccount |
| **Operations** | Service methods | CRUD + Watch |
| **Advantages** | Remote, service operations | Native K8s, efficient watching |

## Next steps

- See `watch-workflow` for real-time progress tracking
- See `workflow-template` for reusable templates
- See `alternate_auth` for production authentication patterns (in-cluster, token-based)
