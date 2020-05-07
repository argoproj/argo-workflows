# Cost Optimisation

## Set Resources Requests Of Your Argo Installation

Use this is you have many installations.

Set a resource quota for the namespace you install Argo in to limit its total usage, e.g.

```
apiVersion: v1
kind: ResourceQuota
metadata:
  name: resource-quota
spec:
  hard:
    pods: "4"
    limits.cpu: 2000m
    limits.memory: 2Gi
    requests.cpu: 1000m
    requests.memory: 1Gi
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
      default:
        cpu: 500m
        memory: 128Mi
      defaultRequest:
        cpu: 100m
        memory: 64Mi
```

This above limit is suitable for the Argo Server, as this is stateless. The Workflow Controller is stateful and will scale to the number of live workflows - so you are likely to need higher values.

## Limit The Total Number Of Workflows And Pods

The workflow controller memory and CPU needs increase linearly with the number of pods and workflows you are currently running. Limit these using TTL

* [Workflow TTL Strategy](fields.md#ttlstrategy) - delete completed workflows after a time
* [Pod GC](fields.md#podgc) - delete completed pods after a time

You can set these configurations globally using [Default Workflow Spec](default-workflow-specs.md).

If you need to keep records historically, use the [Workflow Archive](workflow-archive.md).

## Set Pod Resource Requests 

[Resource duration](resource-duration.md) shows the amount of CPU and memory requested by a pod and is indicative of the cost. You can use this to find costly steps within your workflow.

Smaller requests can be set in the pod spec patch's [resource requirements](fields.md#resourcerequirements). 

