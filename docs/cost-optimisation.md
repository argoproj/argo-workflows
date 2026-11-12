# Cost Optimization

## User Cost Optimizations

Suggestions for users running workflows.

### Set The Workflows Pod Resource Requests

> Suitable if you are running a workflow with many homogeneous pods.

[Resource duration](resource-duration.md) shows the amount of CPU and memory requested by a pod and is indicative of the cost.
You can use this to find costly steps within your workflow.

You can set a smaller value for requests in the pod spec patch's [resource requirements](fields.md#resourcerequirements).

## Use A Node Selector To Use Cheaper Instances

You can use a [node selector](fields.md#nodeselector) for cheaper instances, for example, spot instances:

```yaml
nodeSelector:
  "node-role.kubernetes.io/argo-spot-worker": "true"
```

### Consider trying Volume Claim Templates or Volumes instead of Artifacts

> Suitable if you have a workflow that passes a lot of artifacts within itself.

Copying artifacts to and from storage outside of a cluster can be expensive.
The correct choice depends on the cost of your artifact storage compared with the cost of the volumes you use.
For example, it may be more expensive to allocate and delete a new block storage volume (AWS EBS, GCP persistent disk) every workflow using the PVC feature, than it is to upload and download some small files to object storage (AWS S3, GCP cloud storage).

On the other hand, if you are using an NFS volume shared between all your workflows with large artifacts, that might be cheaper than the data transfer and storage costs of object storage.

Consider:

* Data transfer costs (upload/download compared to copying)
* Data storage costs (object storage compared to volume)
* Requirement for parallel access to data (NFS compared to block storage or artifact)

When using volume claims, consider configuring [Volume Claim GC](fields.md#volumeclaimgc).
By default, Argo only deletes claims when a workflow succeeds.

### Limit The Total Number Of Workflows And Pods

> Suitable for all.

A workflow (and for that matter, any Kubernetes resource) incurs a cost as long as it exists in your cluster, even after it's no longer running.

The workflow controller's memory and CPU needs increase linearly with the number of workflows in your cluster, including completed ones, and the number of pods that haven't completed.

You should delete workflows once you no longer need them.
You can enable the [Workflow Archive](workflow-archive.md) to continue viewing them after they're gone from Kubernetes.

Limit the total number of workflows using:

* Active Deadline Seconds - terminate running workflows that do not complete in a set time.
  This makes sure workflows do not run forever.
* [Workflow TTL Strategy](fields.md#ttlstrategy) - delete completed workflows after a set time.
* [Pod GC](fields.md#podgc) - delete completed pods.
  By default, Argo doesn't delete pods.
* [`CronWorkflow` history limits](cron-workflows.md#cronworkflow-options) - delete successful or failed workflows which exceed the limit.

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

Changing these settings does not delete workflows that have already run.
To list completed workflows created in the last 7 days:

```bash
argo list --completed --since 7d
```

To list/delete workflows completed over 7 days ago:

```bash
argo list --older 7d
argo delete --older 7d
```

## Operator Cost Optimizations

Suggestions for operators who installed Argo Workflows.

### Set Resources Requests and Limits

> Suitable if you have many instances, for example, on dozens of clusters or namespaces.

Set resource requests and limits for the `workflow-controller` and `argo-server`, for example:

```yaml
requests:
  cpu: 100m
  memory: 64Mi
limits:
  cpu: 500m
  memory: 128Mi
```

These values are suitable for the Argo Server, as it is stateless.
The Workflow Controller is stateful and scales to the number of live workflows, so you are likely to need higher values.

### Configure Executor Resource Requests

> Suitable for all - unless you have large artifacts.

Configure [workflow-controller-configmap.yaml](workflow-controller-configmap.yaml) to set the `executor.resources`:

```yaml
executor: |
  resources:
    requests:
      cpu: 100m
      memory: 64Mi
    limits:
      cpu: 500m
      memory: 512Mi
```

The correct values depend on the size of artifacts your workflows download.
For artifacts > 10GB, memory usage may be large - [#1322](https://github.com/argoproj/argo-workflows/issues/1322).
