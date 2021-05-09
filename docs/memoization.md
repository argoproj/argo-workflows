# Step Level Memoization

![beta](assets/beta.svg)

> v2.10 and after

## Introduction

Workflows often have outputs that are expensive to compute. 
This feature reduces cost and workflow execution time by memoizing previously run steps: 
it stores the outputs of a template into a specified cache with a variable key.

## Cache Method

Currently, caching can only be performed with ConfigMaps.
This allows you to easily manipulate cache entries manually through `kubectl` and the Kubernetes API without having to go through Argo.  

## Using Memoization 

Memoization is set at the template level. You must specify a key, which can be static strings but more often depend on inputs. 
You must also specify a name for the ConfigMap cache. 

```
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
        cache:
          configMap:
            name: whalesay-cache

...
```

## FAQs

1. If you see errors like `"error creating cache entry: ConfigMap \"reuse-task\" is invalid: []: Too long: must have at most 1048576 characters"`,
   this is due to [the 1MB limit placed on the size of `ConfigMap`](https://github.com/kubernetes/kubernetes/issues/19781).
   Here are a couple of ways that might help resolve this:
    * Delete the existing `ConfigMap` cache or switch to use a different cache.
    * Reduce the size of the output parameters for the nodes that are being memoized.
    * Split your cache into different memoization keys and cache names so that each cache entry is small.
