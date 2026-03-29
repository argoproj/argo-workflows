# Automatic Pod Restarts

Argo Workflows can automatically restart pods that fail due to infrastructure issues before the main container starts.
This feature handles transient failures like node evictions, disk pressure, or unexpected admission errors without requiring a `retryStrategy` on your templates.

## How It Works

When a pod fails before its main container enters the Running state, the workflow controller checks if the failure reason indicates an infrastructure issue.
If so, the pod is automatically deleted and recreated, allowing the workflow to continue.
For safety this mechanism only works on pods we know never started, for pods that might have started `retryStrategy` is the solution.

This is different from [retryStrategy](retries.md), which handles application-level failures after the container has run.
These are complementary mechanisms, in that both can occur.
Automatic pod restarts handle infrastructure-level failures that occur before your code even starts.

### Restartable Failure Reasons

The following pod failure reasons trigger automatic restarts:

| Reason | Description |
|--------|-------------|
| `Evicted` | Node pressure eviction (`DiskPressure`, `MemoryPressure`, etc.) |
| `NodeShutdown` | Graceful node shutdown |
| `NodeAffinity` | Node affinity/selector no longer matches |
| `UnexpectedAdmissionError` | Unexpected error during pod admission |

### Conditions for Restart

A pod qualifies for automatic restart when ALL of the following are true:

1. The pod phase is `Failed`
2. The main container never entered the `Running` state
3. The failure reason is one of the restartable reasons listed above
4. The restart count for this pod hasn't exceeded the configured maximum

## Configuration

Enable automatic pod restarts in the workflow controller ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  failedPodRestart: |
    enabled: true
    maxRestarts: 3
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | `bool` | `false` | Enable automatic pod restarts |
| `maxRestarts` | `int` | `3` | Maximum restart attempts per node before giving up |

## Monitoring

When a pod is automatically restarted, the node status is updated with:

- `FailedPodRestarts`: Counter tracking how many times the pod was restarted
- `Message`: Updated to indicate the restart, e.g., `Pod auto-restarting due to Evicted: The node had condition: [DiskPressure]`

You can view restart counts in the workflow status:

```bash
kubectl get wf my-workflow -o jsonpath='{.status.nodes[*].failedPodRestarts}'
```

The [`pod_restarts_total`](metrics.md#pod_restarts_total) metric tracks restarts by reason, condition, and namespace.

## Comparison with `retryStrategy`

| Feature | Automatic Pod Restarts | retryStrategy |
|---------|----------------------|---------------|
| **Trigger** | Infrastructure failures before container starts | Application failures after container runs |
| **Configuration** | Global (controller ConfigMap) | Per-template |
| **Use case** | Node evictions, disk pressure, admission errors | Application errors, transient failures |
| **Counter** | `failedPodRestarts` in node status | `retries` in node status |

Both features can work together.
If a pod is evicted before starting, automatic restart handles it.
If the container runs and fails, `retryStrategy` handles it.
Some pods may not be idempotent, and so a `retryStrategy` would not be suitable, but restarting the pod is safe.

## Example

A workflow running on a node that experiences disk pressure:

1. Pod is scheduled and init containers start
2. Node experiences `DiskPressure`, evicting the pod before main container starts
3. Controller detects the eviction and `FailedPodRestarts` condition
4. Pod is deleted, and in workflow the node is marked as Pending to recreate the pod
5. New pod is created on a healthy node
6. Workflow continues normally

The workflow succeeds without any template-level retry configuration needed.
