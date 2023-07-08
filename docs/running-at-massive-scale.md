# Running At Massive Scale

Argo Workflows is an incredibly scalable tool for orchestrating workflows. It empowers you to process thousands of workflows per day, with each workflow consisting of tens of thousands of nodes. Moreover, it effortlessly handles hundreds of thousands of smaller workflows daily. However, optimizing your setup is crucial to fully leverage this capability.

## Run The Latest Version

You must be running at least v3.1 for several recommendations to work. Upgrade to the very latest patch. Performance
fixes often come in patches.

## Test Your Cluster Before You Install Argo Workflows

You'll need a big cluster, with a big Kubernetes master.

Users often encounter problems with Kubernetes needing to be configured for the scale. E.g. Kubernetes API server being
too small. We recommend you test your cluster to make sure it can run the number of pods they need, even before
installing Argo. Create pods at the rate you expect that it'll be created in production. Make sure Kubernetes can keep
up with requests to delete pods at the same rate.

You'll need to GC data quickly. The less data that Kubernetes and Argo deal with, the less work they need to do. Use
pod GC and workflow GC to achieve this.

## Overwhelmed Kubernetes API

Where Argo has a lot of work to do, the Kubernetes API can be overwhelmed. There are several strategies to reduce this:

* Use the Emissary executor (>= v3.1). This does not make any Kubernetes API requests (except for resources template).
* Limit the number of concurrent workflows using parallelism.
* Rate-limit pod creation [configuration](workflow-controller-configmap.yaml) (>= v3.1).
* Set `DEFAULT_REQUEUE_TIME=1m` (see [docs](https://github.com/argoproj/argo-workflows/blob/master/docs/environment-variables.md)).

## Overwhelmed Database

If you're running workflows with many nodes, you'll probably be offloading data to a database. Offloaded data is kept
for 5m. You can reduce the number of records created by setting `DEFAULT_REQUEUE_TIME=1m`. This will slow reconciliation,
but will suit workflows where nodes run for over 1m.

## Miscellaneous

See also [Scaling](scaling.md).
