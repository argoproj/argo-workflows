# Scaling

For running large workflows, you'll typically need to scale the controller to match.

## Horizontally Scaling

You cannot horizontally scale the controller.
The controller supports having a hot-standby for [High Availability](high-availability.md#workflow-controller).

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

The Kubernetes client library used by the Workflow Controller rate limits the number of API requests that can be sent to the Kubernetes API server.
This rate limiting helps prevent overwhelming the API server with too many requests at once.

If you frequently see messages similar to this in the Controller log (issued by the library):

```txt
Waited for 7.090296384s due to client-side throttling, not priority and fairness, request: GET:https://10.100.0.1:443/apis/argoproj.io/v1alpha1/namespaces/argo/workflowtemplates/s2t
```

Or, in >= v3.5, if you see warnings similar to this (could be any CR, not just `WorkflowTemplate`):

```txt
Waited for 7.090296384s, request:GET:https://10.100.0.1:443/apis/argoproj.io/v1alpha1/namespaces/argo/workflowtemplates/s2t
```

These messages indicate that the Controller is being throttled by the client-side rate limiting.

#### Adjusting Rate Limiting

By using cluster-wide observability tooling, you can determine whether or not your Kubernetes API server can handle more requests.
You can increase the rate limits by adjusting the `--qps` and `--burst` arguments for the Controller:

- `--qps`: This argument sets the average number of queries per second allowed by the Kubernetes client.
The default value is 20.
- `--burst`: This argument sets the number of queries per second the client can send before it starts enforcing the qps limit.
The default value is 30. Typically, burst should be greater than qps.

By increasing these values, you can allow the Controller to send more requests to the API server, reducing the likelihood of throttling.

##### Example Configuration

To increase the rate limits, you might set the arguments as follows:

```yaml
args:
  - --qps=50
  - --burst=75
```

This configuration allows the Controller to send an average of 50 queries per second, with a burst capacity of 75 queries per second before throttling is enforced.

It is important to note that increasing these values can increase the load on the Kubernetes API server and that you must observe your Kubernetes API under load in order to determine whether or not the values you have chosen are correct for your needs.
It is not possible to provide a one-size-fits-all recommendation for these values.

## Sharding

### One Install Per Namespace

Rather than running a single installation in your cluster, run one per namespace using the `--namespaced` flag.

### Instance ID

Within a cluster can use instance ID to run N Argo instances within a cluster.

Create one namespace for each Argo, e.g. `argo-i1`, `argo-i2`:.

Edit [`workflow-controller-configmap.yaml`](workflow-controller-configmap.yaml) for each namespace to set an instance ID.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
    instanceID: i1
```

You may need to pass the instance ID to the CLI:

```bash
argo --instanceid i1 submit my-wf.yaml
```

You do not need to have one instance ID per namespace, you could have many or few.

### Maximum Recursion Depth

In order to protect users against infinite recursion, the controller has a default maximum recursion depth of 100 calls to templates.

This protection can be disabled with the [environment variable](environment-variables.md#controller) `DISABLE_MAX_RECURSION=true`

## Miscellaneous

See also [Running At Massive Scale](running-at-massive-scale.md).
