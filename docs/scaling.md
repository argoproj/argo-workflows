# Scaling

For running large workflows, you'll typically need to scale the controller to match.

## Horizontally Scaling

You cannot horizontally scale the controller.

> v3.0

As of v3.0, the controller supports having a hot-standby for [High Availability](high-availability.md#workflow-controller).

## Vertically Scaling

You can scale the controller vertically in these ways:

### Container Resource Requests

If you observe the Controller using its total CPU or memory requests, you should increase those.

### Adding Goroutines to Increase Concurrency

If you have sufficient CPU cores, you can take advantage of them with more goroutines:

- If you have many Workflows and you notice they're not being reconciled fast enough, increase `--workflow-workers`.
- If you're using `TTLStrategy` in your Workflows and you notice they're not being deleted fast enough, increase `--workflow-ttl-workers`.
- If you're using `PodGC` in your Workflows and you notice the Pods aren't being deleted fast enough, increase `--pod-cleanup-workers`.

> v3.5 and after

- If you're using a lot of `CronWorkflows` and they don't seem to be firing on time, increase `--cron-workflow-workers`.

### K8S API Client Side Rate Limiting

The K8S client library rate limits the messages that can go out. The default values are fairly low. If you frequently see a message similar to this in the Controller log (issued by the library):

`Waited for 7.090296384s due to client-side throttling, not priority and fairness, request: GET:https://10.100.0.1:443/apis/argoproj.io/v1alpha1/namespaces/argo/workflowtemplates/s2t`

or for >= v3.5: a warning like this (could be any CR, not just `WorkflowTemplate`):

`Waited for 7.090296384s, request:GET:https://10.100.0.1:443/apis/argoproj.io/v1alpha1/namespaces/argo/workflowtemplates/s2t`

then assuming your K8S API Server can handle it:

- Increase both `--qps` and `--burst`. The `qps` value indicates the average number of queries per second allowed by the K8S Client. The `--burst` value is the number of queries/sec the Client receives before it starts enforcing `qps`, so typically `--burst` > `qps`.

## Sharding

### One Install Per Namespace

Rather than running a single installation in your cluster, run one per namespace using the `--namespaced` flag.

### Instance ID

Within a cluster can use instance ID to run N Argo instances within a cluster.

Create one namespace for each Argo, e.g. `argo-i1`, `argo-i2`:.

Edit [workflow-controller-configmap.yaml](workflow-controller-configmap.yaml) for each namespace to set an instance ID.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
    instanceID: i1
```

> v2.9 and after

You may need to pass the instance ID to the CLI:

```bash
argo --instanceid i1 submit my-wf.yaml
```

You do not need to have one instance ID per namespace, you could have many or few.

## Miscellaneous

See also [Running At Massive Scale](running-at-massive-scale.md).
