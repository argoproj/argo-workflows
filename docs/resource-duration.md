# Resource Duration

![GA](assets/ga.svg)

> v2.7 and after

Argo Workflows provides an indication of how much resource your workflow has used and saves this 
information. This is intended to be an **indicative but not accurate** value.

## Calculation

The calculation is always an estimate, and is calculated by [duration.go](https://github.com/argoproj/argo/blob/master/util/resource/duration.go) 
based on container duration, specified pod resource requests, limits, or (for memory and CPU) 
defaults. 

Each indicator is divided by a common denominator depending on resource type.

### Base Amounts

Each resource type has a denominator used to make large values smaller.

 * CPU: `1`
 * Memory: `1Gi`
 * Storage: `10Gi`
 * Ephemeral Storage: `10Gi`
 * All others: `1` 

The requested fraction of the base amount will be multiplied by the container's run time to get 
the container's Resource Duration. 

For example, if you've requested `100Mi` of memory (one tenth of the base amount), and the container 
runs 120sec, then the reported Resource Duration will be `12sec * (1Gi memory)`. 

### Request Defaults

If `requests` are not set for a container, Kubernetes defaults to `limits`. If `limits` are not set,
Argo falls back to `100m` for CPU and `100Mi` for memory. 

**Note:** these are Argo's defaults, _not_ Kubernetes' defaults. For the most meaningful results, 
set `requests` and/or `limits` for all containers.

### Example

A pod that runs for 3min, with a CPU limit of `2000m`, no memory request and an `nvidia.com/gpu` 
resource limit of `1`:

```
CPU:    3min * 2000m / 1000m = 6min * (1 cpu)
Memory: 3min * 100Mi / 1Gi   = 18sec * (100Mi memory)
GPU:    3min * 1     / 1     = 2min * (1 nvidia.com/gpu)
```

### Web/CLI reporting

Both the web and CLI give abbreviated usage, like `9m10s*cpu,6s*memory,2m31s*nvidia.com/gpu`. In
this context, resources like `memory` refer to the "base amounts".

For example, `memory` means "amount of time a resource requested 1Gi of memory." If a container only 
uses 100Mi, each second it runs will only count as a tenth-second of `memory`.

## Rounding Down

For short running pods (<10s), the memory value may be 0s. This is because the default is `100Mi`, 
but the denominator is `1Gi`. 
