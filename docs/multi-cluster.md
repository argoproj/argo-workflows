# Multi-Cluster

You can execute workflows that run pods in multiple clusters.

This mode only listens to workflows within the controller's cluster. You cannot create workflows in other clusters - you must install a workflow controller in every cluster you create a workflow in.

This feature is orthogonal to managed namespaces. If you install in single-namespace mode but configure multiple clusters - you'll only manage workflow pods in just that namespace of those other clusters.

To manage multiple clusters:

```bash
kubectl -n argo create secret generic clusters
argo cluster add other 
```

Restart the workflow controller.


