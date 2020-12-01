# Cost Optimisation

## User Cost Optimisations

Suggestions for users running workflows.

### Set The Workflows Pod Resource Requests 

> Suitable if you are running a workflow with many homogenous pods.

[Resource duration](resource-duration.md) shows the amount of CPU and memory requested by a pod and is indicative of the cost. You can use this to find costly steps within your workflow.

Smaller requests can be set in the pod spec patch's [resource requirements](fields.md#resourcerequirements). 

## Use A Node Selector To Use Cheaper Instances

You can use a [node selector](fields.md#nodeselector) for cheaper instances, e.g. spot instances:

```yaml
nodeSelector:
  "node-role.kubernetes.io/argo-spot-worker": "true"
```

### Consider trying Volume Claim Templates or Volumes instead of Artifacts

> Suitable if you have a workflow that passes a lot of artifacts within itself.

Copying artifacts to and from storage outside of a cluster can be expensive. The correct choice is dependent on your artifact storage provider is vs. what volume they are using. For example, we believe it may be more expensive to allocate and delete new EBS volumes every workflow using the PVC feature, than it is to upload and download some small files to S3.

On the other hand if they are using a NFS volume shared between all their workflows with large artifacts, that might be cheaper than the data transfer and storage costs of S3.

Consider:

* Data transfer costs (upload/download vs. copying)
* Data storage costs (s3 vs. volume)
* Requirement for parallel access to data (NFS vs. EBS vs. artifact)

### Limit The Total Number Of Workflows And Pods

> Suitable for all.

A workflow (and for that matter, any Kubernetes resource) will incur a cost as long as they exist in your cluster, even after they are no longer running.

The workflow controller memory and CPU needs increase linearly with the number of pods and workflows you are currently running.

You should delete workflows once they are no longer needed, or enable a [Workflow Archive](workflow-archive.md) and you can still view them after they are removed from Kubernetes.

Limit the total number of workflows using:

* Active Deadline Seconds - terminate running workflows that do not complete in a set time. This will make sure workflows do not run forever.
* [Workflow TTL Strategy](fields.md#ttlstrategy) - delete completed workflows after a time
* [Pod GC](fields.md#podgc) - delete completed pods after a time

Example

```yaml
spec:
  # must complete in 8h (28,800 seconds)
  activeDeadlineSeconds: 28800
  # keep workflows for 1d (86,400 seconds)
  ttlStrategy:
    secondsAfterCompletion: 86400
  # delete all pods as soon as they complete
  podGC:
    strategy: OnPodCompletion
```

You can set these configurations globally using [Default Workflow Spec](default-workflow-specs.md).

Changing these settings will not delete workflows that have already run. To list old workflows:

```
argo list --completed --since 7d
```

> v2.9 and after

To list/delete workflows completed over 7 days ago:

```
argo list --older 7d
argo delete --older 7d
```

## Operator Cost Optimisations

Suggestions for operators who installed Argo Workflows.

### Set Resources Requests and Limits

> Suitable if you have many instances, e.g. on dozens of clusters or namespaces.

Set a resource requests and limits for the `workflow-controller` and `argo-server`, e.g. 

```yaml
requests:
  cpu: 100m
  memory: 64Mi
limits:
  cpu: 500m
  memory: 128Mi
```

This above limit is suitable for the Argo Server, as this is stateless. The Workflow Controller is stateful and will scale to the number of live workflows - so you are likely to need higher values.

### Configure Executor Resource Requests

> Suitable for all - unless you have large artifacts.

Configure [workflow-controller-configmap.yaml](workflow-controller-configmap.yaml) to set the `executorResources`

```yaml
executorResources:
  requests:
    cpu: 100m
    memory: 64Mi
  limits:
    cpu: 500m
    memory: 512Mi
```

The correct values depend on the size of artifacts your workflows download. For artifacts > 10GB, memory usage may be large - [#1322](https://github.com/argoproj/argo/issues/1322).
