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
* Set [`DEFAULT_REQUEUE_TIME=1m`](environment-variables.md)

### Argo Server client rate limits

When large workflows are submitted through the Argo Server, its Kubernetes client can throttle
before the cluster does: hydrating stored templates and node status is one API request per
workflow, and with the default `--kube-api-qps=20` / `--kube-api-burst=30` that work is queued
client-side, which shows up as a submission or a retry that appears to stall for minutes while
nothing new is scheduled. For workflows with thousands of nodes, consider raising the server's
limits (`--kube-api-qps=50 --kube-api-burst=100`) and sizing them against your apiserver's own
capacity.

### Bounding server request times

The Argo Server sets no HTTP timeouts by default. If you prefer requests to be bounded, set
`readTimeout` and `writeTimeout` (for example `10m` each) and optionally `idleTimeout`
(for example `15m`) in the controller ConfigMap. These apply to the listener that serves both
HTTP and gRPC, so a read/write timeout also bounds long-lived streaming calls such as
`WatchWorkflows`.

## Overwhelmed Database

If you're running workflows with many nodes, you'll probably be offloading data to a database. Offloaded data is kept
for 5m. You can reduce the number of records created by setting `DEFAULT_REQUEUE_TIME=1m`. This will slow reconciliation,
but will suit workflows where nodes run for over 1m.

## Miscellaneous

See also [Scaling](scaling.md).
