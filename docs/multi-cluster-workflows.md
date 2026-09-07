# Multi-cluster Workflows

> **NOTE**: This document was generated with AI assistance by glm-oss-agent (KR-Ravindra) and includes AI-generated content as permitted by CNCF projects.

You can run Argo Workflows across multiple Kubernetes clusters to:

- Leverage resources from multiple cloud providers or regions
- Improve resilience by distributing workloads across clusters
- Centralize workflow control while executing on multiple data clusters

This guide covers the core multi-cluster features and how to use them together.

## Overview

Argo Workflows provides several mechanisms for running workflows across multiple clusters:

- **ClusterWorkflowTemplates**: Cluster-scoped templates that can be referenced across namespaces and clusters
- **Cross-cluster template references**: Reference cluster templates from workflow templates using `clusterScope: true`
- **Workflow templates as workflows**: Submit a workflow directly from a cluster workflow template
- **Synchronization**: Manage parallel execution limits across multiple workflow controllers using mutexes or semaphores

## ClusterWorkflowTemplates

ClusterWorkflowTemplates are cluster-scoped WorkflowTemplates that can be accessed across all namespaces in a cluster. They are useful when you want to share common workflow templates across multiple teams or namespaces.

### Creating a ClusterWorkflowTemplate

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-print-message
spec:
  templates:
  - name: print-message
    inputs:
      parameters:
      - name: message
    container:
      image: busybox
      command: [echo]
      args: ["{{inputs.parameters.message}}"]
```

Apply the template:

```bash
kubectl apply -f cluster-print-message.yaml
```

### Using a ClusterWorkflowTemplate in a Workflow

You can reference a ClusterWorkflowTemplate from a Workflow using `templateRef` with `clusterScope: true`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: hello-world
  templates:
  - name: hello-world
    steps:
    - - name: call-cluster-template
        templateRef:
          name: cluster-print-message
          template: print-message
          clusterScope: true
        arguments:
          parameters:
          - name: message
            value: "Hello from multi-cluster workflow!"
```

### Submitting a Workflow from a ClusterWorkflowTemplate

You can create a workflow directly from a ClusterWorkflowTemplate that has an `entrypoint` defined:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  workflowTemplateRef:
    name: cluster-print-message
    clusterScope: true
```

Apply the workflow:

```bash
kubectl apply -f hello-world.yaml
```

or using the CLI:

```bash
argo submit --from clusterworkflowtemplate/cluster-print-message
```

### Referencing Other ClusterWorkflowTemplates

You can reference templates from other ClusterWorkflowTemplates within the same template using `templateRef` with `clusterScope: true`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-print-message
spec:
  templates:
  - name: print-message
    inputs:
      parameters:
      - name: message
    container:
      image: busybox
      command: [echo]
      args: ["{{inputs.parameters.message}}"]
  - name: print-steps
    steps:
    - - name: call-cluster-template
        templateRef:
          name: cluster-print-message
          template: print-message
          clusterScope: true
        arguments:
          parameters:
          - name: message
            value: "Step 1"
    - - name: call-cluster-template
        templateRef:
          name: cluster-print-message
          template: print-message
          clusterScope: true
        arguments:
          parameters:
          - name: message
            value: "Step 2"
```

## Running Workflows Across Multiple Clusters

### Prerequisites

1. Argo Workflows installed in each cluster
2. Argo CLI configured for each cluster
3. Network connectivity between clusters (for artifact repositories, service discovery, etc.)

### Using ClusterWorkflowTemplates Across Clusters

ClusterWorkflowTemplates are cluster-scoped, so they exist only within the cluster where they're defined. To use the same template across multiple clusters:

1. Define the ClusterWorkflowTemplate in each cluster
2. Point the template to resources available in that cluster
3. Reference the template using the same name

```bash
# Define template in cluster A
kubectl apply -f - <<EOF
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-print-message
spec:
  templates:
  - name: print-message
    container:
      image: busybox
      command: [echo]
      args: ["Message from cluster A"]
EOF

# Define template in cluster B
kubectl apply -f - <<EOF
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-print-message
spec:
  templates:
  - name: print-message
    container:
      image: busybox
      command: [echo]
      args: ["Message from cluster B"]
EOF

# Submit workflow in cluster A referencing cluster template
kubectl apply -f - <<EOF
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  workflowTemplateRef:
    name: cluster-print-message
    clusterScope: true
EOF
```

### Using Different Artifact Repositories Across Clusters

When running workflows across clusters, you may need different artifact repositories:

1. Define artifact repositories in each cluster's workflow-controller-configmap
2. Reference the repository using the repository name

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: artifacts-cross-cluster-
spec:
  arguments:
    artifacts:
    - name: input-file
      path: /tmp/input.txt
      # Referenced artifact repository (see workflow-controller-configmap)
      artifactRepository:
        s3:
          endpoint: s3.amazonaws.com
          bucket: my-cluster-a-bucket
  templates:
  - name: process-artifact
    inputs:
      artifacts:
      - name: input-file
        path: /tmp/input.txt
    container:
      image: busybox
      command: [sh, -c, "cat /tmp/input.txt"]
```

## Synchronization Across Multiple Controllers

When you have multiple workflow controllers (e.g., across different clusters), you can use synchronization to manage parallel execution limits.

### Using ConfigMap-based Semaphores

ConfigMap-based semaphores are shared across all controllers in the same cluster. For true cross-cluster synchronization, use database-based semaphores.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: sync-configmap-
spec:
  synchronization:
    semaphores:
    - configMapKeyRef:
        key: max-workflows
        name: workflow-limits
  entrypoint: hello-world
  templates:
  - name: hello-world
    container:
      image: busybox
      command: [sh, -c, "echo 'Running with sync limit'"]
```

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-limits
data:
  max-workflows: "10"  # Only 10 workflows can run concurrently
```

### Using Database-based Semaphores for Cross-Cluster Limits

For true cross-cluster synchronization, configure a shared database:

1. Set up a PostgreSQL, MySQL, or MariaDB database
2. Configure it in the workflow-controller-configmap in each cluster
3. Use `database: true` in your synchronization configuration

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: sync-database-
spec:
  synchronization:
    mutexes:
    - database: true
      name: critical-section
  entrypoint: hello-world
  templates:
  - name: hello-world
    container:
      image: busybox
      command: [sh, -c, "echo 'Critical section'"]
```

The synchronization API can be enabled to manage semaphore limits programmatically:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  config: |
    synchronization:
      enableAPI: true
      # Database configuration for cross-cluster sync
      sharedStateConfigMap:
        key: synchronization-state
        name: global-sync-state
```

See [Synchronization](synchronization.md) for more details.

## Common Patterns

### Workflow with Mixed Cluster and Namespaced Templates

You can use a mix of cluster and namespaced templates in a single workflow:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: mixed-cluster-wf-
spec:
  entrypoint: steps
  templates:
  - name: steps
    steps:
    # Use namespaced workflow template
    - - name: local-task
        template: local-task
    
    # Use cluster workflow template
    - - name: cluster-task
        templateRef:
          name: cluster-print-message
          template: print-message
          clusterScope: true
        arguments:
          parameters:
          - name: message
            value: "Cluster task"
```

### Diamond Workflow with Cluster Templates

You can create complex DAG workflows using cluster templates:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: diamond-cluster-
spec:
  entrypoint: diamond
  templates:
  - name: diamond
    dag:
      tasks:
      - name: A
        templateRef:
          name: cluster-print-message
          template: print-message
          clusterScope: true
        arguments:
          parameters:
          - name: message
            value: "A"
      - name: B
        depends: "A"
        templateRef:
          name: cluster-print-message
          template: print-message
          clusterScope: true
        arguments:
          parameters:
          - name: message
            value: "B"
      - name: C
        depends: "A"
        templateRef:
          name: cluster-print-message
          template: print-message
          clusterScope: true
        arguments:
          parameters:
          - name: message
            value: "C"
      - name: D
        depends: "B && C"
        templateRef:
          name: cluster-print-message
          template: print-message
          clusterScope: true
        arguments:
          parameters:
          - name: message
            value: "D"
```

## Comparison: Cluster vs Namespaced Templates

| Feature | ClusterWorkflowTemplate | WorkflowTemplate |
| --------- | ------------------------ | ------------------ |
| Scope | Cluster-scoped | Namespace-scoped |
| Access | All namespaces in cluster | Only defined namespace |
| Sharing | Across teams/namespaces | Limited to namespace |
| Template References | Can reference other ClusterWorkflowTemplates | Can reference only namespace templates |
| Best For | Common templates across teams, centralized CI/CD | Team-specific templates |

## CLI Usage

### Creating a ClusterWorkflowTemplate

```bash
argo cluster-template create https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/cluster-workflow-template/clustertemplates.yaml
```

### Listing ClusterWorkflowTemplates

```bash
kubectl get clusterworkflowtemplates
argo cluster-template list
```

### Viewing ClusterWorkflowTemplate Details

```bash
kubectl get clusterworkflowtemplate cluster-print-message -o yaml
argo cluster-template get cluster-print-message
```

### Deleting a ClusterWorkflowTemplate

```bash
kubectl delete clusterworkflowtemplate cluster-print-message
argo cluster-template delete cluster-print-message
```

## Getting Help

- Read the [Cluster Workflow Templates](cluster-workflow-templates.md) reference
- Review [Synchronization](synchronization.md) for concurrency management
- See [Synchronization Config](synchronization-config.md) for API usage
- Check the [Examples](../examples/cluster-workflow-template/) directory for runnable examples
