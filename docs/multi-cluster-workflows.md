# Multi-cluster Workflows

> **NOTE**: This document was generated with AI assistance by glm-oss-agent (KR-Ravindra) and includes AI-generated content as permitted by CNCF projects.

You can run Argo Workflows across multiple Kubernetes clusters to:

- Leverage resources from multiple cloud providers or regions
- Improve resilience by distributing workloads across clusters
- Centralize workflow control while executing on multiple data clusters

This guide covers the core multi-cluster features. See [Multi-cluster Workflows Advanced](multi-cluster-workflows-advanced.md) for synchronization, advanced patterns, and CLI usage.

## Overview

Argo Workflows provides several mechanisms for running workflows across multiple clusters:

- **ClusterWorkflowTemplates**: Cluster-scoped templates that can be referenced across namespaces and clusters
- **Cross-cluster template references**: Reference cluster templates from workflow templates using `clusterScope: true`
- **Workflow templates as workflows**: Submit a workflow directly from a cluster workflow template

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

## Comparison: Cluster vs Namespaced Templates

| Feature | ClusterWorkflowTemplate | WorkflowTemplate |
| --------- | ------------------------ | ------------------ |
| Scope | Cluster-scoped | Namespace-scoped |
| Access | All namespaces in cluster | Only defined namespace |
| Sharing | Across teams/namespaces | Limited to namespace |
| Template References | Can reference other ClusterWorkflowTemplates | Can reference only namespace templates |
| Best For | Common templates across teams, centralized CI/CD | Team-specific templates |

See [Multi-cluster Workflows Advanced](multi-cluster-workflows-advanced.md) for more details.
