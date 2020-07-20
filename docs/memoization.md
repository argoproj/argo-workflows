# Step Level Memoization

![alpha](assets/alpha.svg)

> v2.10 and after

## Introduction

Often a workflow has outputs that are expensive to compute. 
This feature reduces cost and workflow execution time by memoizing previously run steps: 
it stores the outputs of a template into a specified cache with a specified key.

## Caching

Currently, caching can only be performed with ConfigMaps.
This allows you to easily manipulate cache entries manually through `kubectl` and the Kubernetes API without having to go through Argo.  



