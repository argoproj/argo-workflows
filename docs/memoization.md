# Step Level Memoization

> v2.10 and after

## Introduction

Workflows often have outputs that are expensive to compute.
This feature reduces cost and workflow execution time by memoizing previously run steps:
it stores the outputs of a template into a specified cache with a variable key.

## Cache Method

Currently, caching can only be performed with config-maps.
This allows you to easily manipulate cache entries manually through `kubectl` and the Kubernetes API without having to go through Argo.
All cache config-maps must have the label `workflows.argoproj.io/configmap-type: Cache` to be used as a cache. This prevents accidental access to other important config-maps in the system

## Using Memoization

Memoization is set at the template level. You must specify a `key`, which can be static strings but more often depend on inputs.
You must also specify a name for the `config-map` cache.
Optionally you can set a `maxAge` in seconds or hours (e.g. `180s`, `24h`) to define how long should it be considered valid. If an entry is older than the `maxAge`, it will be ignored.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
   generateName: memoized-workflow-
spec:
   entrypoint: whalesay
   templates:
      - name: whalesay
        memoize:
           key: "{{inputs.parameters.message}}"
           maxAge: "10s"
           cache:
              configMap:
                 name: whalesay-cache
```

[Find a simple example for memoization here](https://github.com/argoproj/argo-workflows/blob/master/examples/memoize-simple.yaml).

!!! Note
    In order to use memoization it is necessary to add the verbs `create` and `update` to the `configmaps` resource for the appropriate (cluster) roles. In the case of a cluster install the `argo-cluster-role` cluster role should be updated, whilst for a namespace install the `argo-role` role should be updated.

## FAQ

1. If you see errors like `error creating cache entry: ConfigMap \"reuse-task\" is invalid: []: Too long: must have at most 1048576 characters`,
   this is due to [the 1MB limit placed on the size of `ConfigMap`](https://github.com/kubernetes/kubernetes/issues/19781).
   Here are a couple of ways that might help resolve this:
    * Delete the existing `ConfigMap` cache or switch to use a different cache.
    * Reduce the size of the output parameters for the nodes that are being memoized.
    * Split your cache into different memoization keys and cache names so that each cache entry is small.
