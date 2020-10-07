# Resources Usage

> v2.13 and after

Argo Workflows provides an indication of how much  resource your workflow has used and saves this 
information. This is an **indicative but not accurate** value.

This is taken just once during the lifetime of each pod, just after it is running, rather than averaged over time (which would give a much more accurate figure). This will be accurate if the pod has high resource usage early on it.

The value for resources usage will be blank if:

* The pod was short-lived (<30s) - the Metrics Server will not have had time to capture metrics.
* The workflow controller's role (typically named `argo`) does not have the correct permission, [example](manifests/namespace-install/workflow-controller-rbac/workflow-controller-role.yaml).
* You have not installed the [Metrics Server](https://github.com/kubernetes-sigs/metrics-server).
 
