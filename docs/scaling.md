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
The default value is 30.
Typically, burst should be greater than qps.

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

### Rate Limiting Pod Creation

The Workflow Controller is responsible for the requesting the creation of Pods from the Kubernetes API.
Creating pods is an expensive/heavy operation, and requesting too many Pods at once can in turn cause the Kubernetes API server to become overwhelmed.
To mitigate this, you can set rate limit how many Pods the Workflow Controller requests by adjusting the `limit` and `burst` values in the [Workflow Controller ConfigMap](workflow-controller-configmap.yaml).

```yaml
  # Globally limits the rate at which pods are created.
  # This is intended to mitigate flooding of the Kubernetes API server by workflows with a large amount of
  # parallel nodes.
  resourceRateLimit: |
    limit: 10
    burst: 25
```

- `limit`: This sets the average number of Pod creation requests per second..
- `burst`: This sets the number of Pods per second the Controller can create before it starts enforcing the `limit`.
Typically, burst should be greater than the limit.

By using cluster-wide observability tooling, you can determine whether or not your Kubernetes API server can handle more Pod creation requests.
It is important to note that increasing these values can increase the load on the Kubernetes API server and that you must observe your Kubernetes API under load in order to determine whether or not the values you have chosen are correct for your needs.
It is not possible to provide a one-size-fits-all recommendation for these values.

!!! Note
    Despite the name, this rate limit only applies to the creation of Pods and not the creation of other Kubernetes resources (for example, ConfigMaps or PersistentVolumeClaims).

## Sharding

### One Install Per Namespace

Rather than running a single installation in your cluster, run one per namespace using the `--namespaced` flag.

### Instance ID

Within a cluster you can use instance ID to run many Argo instances within a cluster.
You can run each instance in a separate, or the same namespace.

For each instance, edit the [`workflow-controller-configmap.yaml`](workflow-controller-configmap.yaml) to set an `instanceID`.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
    instanceID: i1
```

#### CLI Usage

When you are using the CLI, set the `--instanceid` to interact with a specific instance.
For example, to submit a Workflow defined in `my-wf.yaml`:

```bash
argo --instanceid i1 submit my-wf.yaml
```

#### Declarative Usage

You can also declare which instance to use in the each of the Custom Resources. For example, on a `Workflow`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
    labels:
        workflows.argoproj.io/controller-instanceid: i1
```

It can also be useful, when defining a `WorkflowTemplate` to use `templateDefaults`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
    labels:
        workflows.argoproj.io/controller-instanceid: i1
    name: example
spec:
    templateDefaults:
        metadata:
            labels:
                workflows.argoproj.io/controller-instanceid: i1
```

### Maximum Recursion Depth

In order to protect users against infinite recursion, the controller has a default maximum recursion depth of 100 calls to templates.

This protection can be disabled with the [environment variable](environment-variables.md#controller) `DISABLE_MAX_RECURSION=true`

### Caching Semaphore Limit ConfigMap Requests

By default the controller will reload the ConfigMap(s) referenced by a semaphore from kube every time that workflow is queued. If you notice high latency from queuing workflows leveraging semaphores you can cache semaphore limits by editing the `semaphoreLimitCacheSeconds` parameter in [`workflow-controller-configmap.yaml`](workflow-controller-configmap.yaml).

Note that this will mean that Argo will not immediately pick up changes to your config map limits.

## Miscellaneous

See also [Running At Massive Scale](running-at-massive-scale.md).
