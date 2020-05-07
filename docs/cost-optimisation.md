# Cost Optimisation

## Limit The Total Number Of Workflows And Pods

> Suitable for all.

The workflow controller memory and CPU needs increase linearly with the number of pods and workflows you are currently running. Limit these using TTL

* [Workflow TTL Strategy](fields.md#ttlstrategy) - delete completed workflows after a time
* [Pod GC](fields.md#podgc) - delete completed pods after a time

You can set these configurations globally using [Default Workflow Spec](default-workflow-specs.md).

If you need to keep records historically, use the [Workflow Archive](workflow-archive.md).

## Set Resources Requests Of Your Argo Instances

> Suitable if you have many instances, e.g. on dozens of clusters or namespaces.

Set a resource quota for the namespace you install Argo in to limit its total usage, e.g.

```
apiVersion: v1
kind: ResourceQuota
metadata:
  name: resource-quota
spec:
  hard:
    pods: "4"
    limits.cpu: 1000m
    limits.memory: 1Gi
    requests.cpu: 500m
    requests.memory: 515Mi
```

Use limit range to set default container requests and limits, e.g.

```
apiVersion: v1
kind: LimitRange
metadata:
  name: limit-range
spec:
  limits:
    - type: Container
      defaultRequest:
        cpu: 100m
        memory: 64Mi
      default:
        cpu: 500m
        memory: 128Mi
```

This above limit is suitable for the Argo Server, as this is stateless. The Workflow Controller is stateful and will scale to the number of live workflows - so you are likely to need higher values.

## Configure Executor Resource Requests

> Suitable for all - unless you have large artifacts.

Configure [workflow-controller-configmap.yaml](workflow-controller-configmap.yaml) to set the `executorResources`

```cgo
    executorResources:
      requests:
        cpu: 100m
        memory: 64Mi
      limits:
        cpu: 500m
        memory: 512Mi
```

The correct values depend on the size of artifacts your workflows download. For artifacts >10GB, memory usage maybe large - [#1322](https://github.com/argoproj/argo/issues/1322).

## Set The Workflows Pod Resource Requests 

> Suitable if you are running a workflow with many homogenous pods.

[Resource duration](resource-duration.md) shows the amount of CPU and memory requested by a pod and is indicative of the cost. You can use this to find costly steps within your workflow.

Smaller requests can be set in the pod spec patch's [resource requirements](fields.md#resourcerequirements). 

