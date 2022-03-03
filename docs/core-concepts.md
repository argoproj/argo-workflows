# Core Concepts

!!! note
    Please read [Kubernetes concepts](https://kubernetes.io/docs/concepts/) first. 

* **Workflow**: a Kubernetes resource defining the execution of one or more **template**. Workflows are named. 
* **Template**: a **step**, **steps** or **dag**. 
* **Step**: a single step of a **workflow**, typically run a container based on **inputs** and capture the **outputs**. 
* **Steps**: a list of **steps**
* **Entrypoint**: the first **step** to execute when running a **workflow** 
* **Node**: a step
* **Directed Acyclic Graph (DAG)**: a set of **steps** (nodes) and the dependencies (edges) between them.
* **Workflow Template**: a Kubernetes resource defining a reusable workflow for a namespace
* **Cluster Workflow Template**: a Kubernetes resource defining a reusable workflow  for a cluster
* **Inputs**: **parameters** and **artifacts** passed to the **step**,
* **Outputs**: **parameters** and **artifacts** outputted by a **step**
* **Parameters**: objects, strings, booleans, arrays
* **Artifacts**: files saved by a container
* **Artifact Repository**: a place where **artifacts** are stored
* **Executor**: the method to execute a container, e.g. Docker, PNS ([learn more](workflow-executors.md))
* **Workflow Service Account**: the service account that a workflow is executed as ([learn more](service-accounts.md))
