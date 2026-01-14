
Description: Support for global agent pod that can execute tasks from multiple workflows using label selectors
Authors: [Gaurang Mishra](https://github.com/gaurang9991)
Component: General
Issues: 7891

<!-- markdownlint-disable MD022 -->
<!-- markdownlint-disable MD031 -->
<!-- markdownlint-disable MD007 -->
<!-- markdownlint-disable MD023 -->
### Overview

This feature enables a single global agent pod per service account to execute tasks from multiple workflows.
Instead of creating one agent pod per workflow, users can opt-in to a shared agent pod model that reduces resource overhead while maintaining security boundaries.
The agent pod uses label selectors to watch and process WorkflowTaskSets from multiple workflows, the configuration also allows users to fully control the life cycle of the agent pod.

### When to Use This Feature

You should enable global agent pods when:

- You run many concurrent workflows with HTTP or plugin templates in the same namespace
- You want to reduce resource overhead by sharing agent pods across workflows
- You want centralized management and monitoring of agent execution
- You need to scale agent workers independently of individual workflows
- You want better resource utilization without sacrificing security isolation per service account

You should continue using per-workflow agents (default) when:

  - You need strict pod-level isolation between workflows
  - You want automatic agent pod cleanup via owner references
  - You have few concurrent workflows with agent requirements
  - You prefer simpler debugging with dedicated agent pods per workflow

### How to Enable Global Agent Mode

#### Step 1: Update ConfigMap

Add the `agent` configuration section to your `workflow-controller-configmap`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
  namespace: argo
data:
  agent: |
    # Enable global agent pod mode (one per service account)
    runMultipleWorkflow: true
    
    # Configure agent pod resources (optional)
    resources:
      requests:
        cpu: 10m
        memory: 64Mi
      limits:
        cpu: 100m
        memory: 256Mi
    
    # Configure security context (optional)
    securityContext:
      runAsNonRoot: true
      runAsUser: 8737
      capabilities:
        drop:
          - ALL
    
    # Keep agent pod alive after all workflows complete (optional)
    deleteAfterCompletion: false
    
    # Let controller create agent pods (optional, default: true)
    createPod: true
```

#### Step 2: Restart Workflow Controller

After updating the ConfigMap, restart the workflow controller to apply the changes:

```bash
kubectl rollout restart deployment workflow-controller -n argo
```

#### Step 3: Run Your Workflows

No changes needed to your workflow definitions.
The controller will automatically create global agent pods as needed when workflows with HTTP or plugin templates are submitted.

### What to Expect

#### Agent Pod Naming

**Per-Workflow Mode (default):**

```text
my-workflow-agent
another-workflow-agent
```

**Global Agent Mode:**

```text
argo-agent-default          # For workflows using 'default' SA
argo-agent-custom-sa        # For workflows using 'custom-sa' SA
```

#### Agent Pod Lifecycle

**Per-Workflow Mode:**

- Agent pod created when workflow starts
- Agent pod deleted when workflow completes (via owner reference)
- Automatic cleanup with workflow

**Global Agent Mode:**

  - Agent pod created when first workflow with a service account needs it
  - Agent pod processes tasks from all workflows using the same service account
  - Agent pod remains running across multiple workflow executions
  - Agent pod deleted only when: `deleteAfterCompletion: true` is set and no active workflows need the agent

#### Task Processing

The global agent pod:

- Watches WorkflowTaskSets using label selector `workflows.argoproj.io/workflow-service-account={sa-name}`
- Extracts workflow UID dynamically from each TaskSet's owner references
- Processes tasks from multiple workflows concurrently using worker goroutines
- Maintains correct workflow context for each task
- Patches results back to the appropriate WorkflowTaskSet

#### Security Isolation

Service account boundaries are maintained:

- Each service account gets its own dedicated agent pod
- Agent pods only process TaskSets for their assigned service account
- No cross-service-account task execution
- RBAC policies apply per service account as before

### Configuration Options

#### `runMultipleWorkflow`

- **Default:** `false`
- **Description:** Enable global agent pod mode
- **Values:** `true` (global mode), `false` (per-workflow mode)

#### `deleteAfterCompletion`

- **Default:** `true`
- **Description:** Delete agent pod when no workflows need it
- **Values:** `true` (auto-cleanup), `false` (keep alive)
- **Note:** Only applies in global mode; per-workflow mode always deletes via owner reference

#### `createPod`

- **Default:** `true`
- **Description:** Controller creates and manages agent pods
- **Values:** `true` (managed), `false` (external)
- **Use Case:** Set to `false` if using external operator or manual agent deployment

#### `resources`

- **Default:** `requests: {cpu: 10m, memory: 64Mi}, limits: {cpu: 100m, memory: 256Mi}`
- **Description:** Resource requests and limits for agent pod main container
- **Configurable:** Both requests and limits for CPU and memory

#### `securityContext`

- **Default:** `runAsNonRoot: true, runAsUser: 8737, capabilities: {drop: [ALL]}`
- **Description:** Security context for agent pod main container
- **Configurable:** All SecurityContext fields supported by Kubernetes

### Migration from Per-Workflow to Global Mode

#### Zero-Downtime Migration

1. Update ConfigMap with `runMultipleWorkflow: true`
2. Restart workflow controller
3. Existing per-workflow agents continue running
4. New workflows use global agent pods
5. Old per-workflow agents clean up naturally as workflows complete

#### Rollback

To rollback to per-workflow mode:

1. Update ConfigMap with `runMultipleWorkflow: false` (or remove the setting)
2. Restart workflow controller
3. Existing global agents remain but become unused
4. New workflows create per-workflow agents
5. Manually delete unused global agents: `kubectl delete pod -l workflows.argoproj.io/agent-service-account -n argo`

### Monitoring and Observability

#### Agent Pod Labels

Global agent pods have these labels:

```yaml
workflows.argoproj.io/agent-service-account: "default"
workflows.argoproj.io/component: "agent"
```

#### Viewing Active Agent Pods

```bash
### List all agent pods
kubectl get pods -l workflows.argoproj.io/component=agent -n argo

### List global agent pods only
kubectl get pods -l workflows.argoproj.io/agent-service-account -n argo
```

#### Agent Pod Logs

```bash
### View logs for a specific service account's agent
kubectl logs argo-agent-default -n argo

### Follow logs in real-time
kubectl logs -f argo-agent-default -n argo
```

#### Checking Agent Configuration

```bash
### View current agent configuration
kubectl get configmap workflow-controller-configmap -n argo -o jsonpath='{.data.agent}' | yq
```

### Troubleshooting

#### Agent Pod Not Created

**Check configuration:**

```bash
kubectl get configmap workflow-controller-configmap -n argo -o yaml
```

**Check controller logs:**

```bash
kubectl logs -l app=workflow-controller -n argo | grep agent
```

#### Tasks Not Processing

**Verify TaskSet labels:**

```bash
kubectl get workflowtasksets -n argo --show-labels
```

**Check agent pod environment:**

```bash
kubectl exec argo-agent-default -n argo -- env | grep ARGO
```

Expected: `ARGO_AGENT_LABEL_SELECTOR=workflows.argoproj.io/workflow-service-account=default`

#### Multiple Agent Pods for Same Service Account

This indicates spec changes (plugin configuration or ConfigMap updates).
The controller creates a new agent pod and the old one will be cleaned up after existing tasks complete.

### Benefits

- **Reduced Resource Usage:** One agent pod per service account instead of per workflow
- **Better Scalability:** Configure worker count once, applies to all workflows
- **Simplified Operations:** Fewer pods to manage, monitor, and troubleshoot
- **Flexible Deployment:** Choose between global or per-workflow agents based on requirements
- **Backward Compatible:** Default behavior unchanged, opt-in for global mode
- **Security Maintained:** Service account boundaries preserved

### Technical Details

#### Code Changes

- `workflow/executor/agent.go` - Dynamic UID extraction and label selector support
- `cmd/argoexec/commands/agent.go` - Removed static UID environment variable
- `workflow/controller/agent.go` - Global agent pod creation and lifecycle management
- `workflow/controller/taskset.go` - Added service account labels to TaskSets
- `workflow/controller/operator.go` - Added cleanup on workflow completion
- `config/config.go` - Added AgentConfig structure

#### Label Selector Logic

The agent watches TaskSets with label: `workflows.argoproj.io/workflow-service-account={service-account-name}`

Each workflow's TaskSet is labeled with:

- `workflows.argoproj.io/workflow-service-account` - For agent filtering
- `workflows.argoproj.io/workflow-name` - For debugging and monitoring
