# Multi-cluster

## Core Concepts

When running workflows that creates resources (i.e. run tasks/steps) in other clusters and namespaces.

* The **primary cluster** is where you'll create your workflows in. All cluster must be given a unique name. In examples
  we'll call this `cluster-0`.
* The **primary namespace** is where workflow is, which may be different to the resource's namespace. In the
  examples, `argo`.
* The **remote cluster** is where the workflow may create pods. In the examples, `cluster-1`.
* The **remote namespace** is where remote resources are created. In the examples, `default`.
* A **profile** is a configuration profile used to connect to a remote cluster.
