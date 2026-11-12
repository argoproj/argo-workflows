# Running At Massive Scale

Argo Workflows is an incredibly scalable tool for orchestrating workflows.
It empowers you to process thousands of workflows per day, with each workflow consisting of tens of thousands of nodes.
Moreover, it effortlessly handles hundreds of thousands of smaller workflows daily.
However, optimizing your setup is crucial to fully leverage this capability.

## Run The Latest Version

Upgrade to the very latest patch.
Performance fixes often come in patches.

## Test Your Cluster Before You Install Argo Workflows

To run Argo Workflows at scale, you need a big cluster, with a big Kubernetes master.

Users often encounter problems with Kubernetes needing to be configured for the scale.
For example, the Kubernetes API server might be too small.
Test your cluster to make sure it can run the number of pods you need, even before installing Argo.
Create pods at the rate you expect in production.
Make sure Kubernetes can keep up with requests to delete pods at the same rate.

You need to garbage collect (GC) data quickly.
The less data that Kubernetes and Argo deal with, the less work they need to do.
Use pod GC and workflow GC to achieve this.

## Overwhelmed Kubernetes API

Where Argo has a lot of work to do, the Kubernetes API can be overwhelmed.
There are several strategies to reduce this:

* Limit the number of concurrent workflows using parallelism.
* Rate-limit pod creation [configuration](workflow-controller-configmap.yaml).
* Set [`DEFAULT_REQUEUE_TIME=1m`](environment-variables.md)

## Overwhelmed Database

If you're running workflows with many nodes, you're probably offloading data to a database.
Offloaded data is kept for 5m.
You can reduce the number of records created by setting `DEFAULT_REQUEUE_TIME=1m`.
This slows reconciliation, but suits workflows where nodes run for over 1m.

## Miscellaneous

See also [Scaling](scaling.md).
