# Resource Duration

![alpha](assets/alpha.svg)

Argo Workflows provides an indication of how much resource your workflow has used and save this information. This is intended to be an **indicative but not accurate** value. 

## Calculation

The calculation is always an estimate, and is calculated by [../util/resource/duration.go](../util/resource/duration.go) based on container duration, specified pod resource requests, limits, or defaults (for memory and CPU). 

Each indicator is divided by a common denominator depending or resource type.

### Example

A pod that runs for 3m, with a CPU limit of 2000m, no memory request and an `nvidia.com/gpu` resource limit of 1:

* CPU: 3 * 60s * 2000m / 1000m = 6s*cpu
* Memory: 3 * 60s * 100m / 1Gi = 0s*memory 
* GPU: 2 * 60s * = 2s*nvidia.com/gpu

## Rounding Down

For short running pods (<10s), the memory value maybe 0s. This is because the default is 100m, but the denominator is 1000m. 