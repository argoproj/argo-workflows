# Usage

![alpha](assets/alpha.svg)

Argo Workflows estimates how much resource your workflow has used and save this information. This is known as "resource usage" and is intended as an **indicative but not accurate** value. 

## Calculation

Usage calculation is always an estimate, and is calculated by [../util/usage/estimator.go](../util/usage/estimator.go) based on container duration, specified pod resource requests, limits, or defaults (for memory and CPU). 

Usage is divided by a common denominator depending or resource type.

### Example

A pod that runs for 3m, with a CPU limit of 2000m, no memory request and an `nvidia.com/gpu` resource limit of 1:

* CPU usage: 3 * 60s * 2000m / 1000m = 6m*cpu
* Memory usage: 3 * 60s * 100m / 1Gi = 0s*memory
* GPU usage: 2 * 60s * = 3m*nvidia.com/gpu

## Limitations & Assumptions

To calculate the usage we assume that request/limit/default for a resource is a good enough representative of the pods average usage.

This is **never** actually the case:

* The pod will probably use more that the request and less than the limit.
* The pod may use more than the limit or less than the request.

This is why the usage is **indicative but not accurate**.

## Memory Usage Truncation

For short running pods (<10s), the memory value maybe 0s. This is because the default is 100m, but the denominator is 1000m. 