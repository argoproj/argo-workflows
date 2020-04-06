# Usage 

![alpha](assets/alpha.svg)

If you have the Metrics Server running, you can captures the pods CPU and memory usage too.

Before you start, read you should read about [resources duration](resource-duration.md) first. **resource duration** saves your requested usage where as **usage** saves as estimate of the actual duration. Together are intended to allow you to estimate **utilization** and therefore size your pods correctly to reduce your costs.   

## Configuration

This is turned on only if [the `usageCapture` flag is enabled in config](workflow-controller-configmap.yaml).

If this is enabled, but the Metrics Service is not available, an error will be printed to the logs every 30s until it is.

You can check to see if the Metrics Service is installed by running:

```
kubectl get podmetrics
``` 

## Capture

Every 30s the workflow controller gets the CPU and memory usage for all running pods, essentially:

For any  pods that are part of a workflow, a usage sample of the CPU and memory usage is taken and added to a the pod's moving average, which is stored in an in-memory cache.

When the pod completes, this value is saved.

Finally, when the workflow completes, the value is aggregated for all pods.

### Why Didn't I Get Any Data

Pods that run shorted than 60s are unlikely to to be sampled.

## Finding Over-Provisioned Workflows

TODO