# Usage 

![alpha](assets/alpha.svg)

If you have the Metrics Server running, you can captures the pods CPU and memory usage too.

Before you start, read you should read about [resources duration](resource-duration.md) first. **resource duration** saves your requested usage where as **usage** saves as estimate of the actual duration. Together are intended to allow you to estimate **utilization** and therefore size your pods correctly to reduces your costs.   

## Configuration

This is turned on only if [the `usageCapture` flag is enabled in config](workflow-controller-configmap.yaml).
