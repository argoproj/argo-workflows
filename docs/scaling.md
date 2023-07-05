# Scaling

For running large workflows, you'll typically need to scale the controller to match.

## Horizontally Scaling

You cannot horizontally scale the controller.

> v3.0

As of v3.0, the controller supports having a hot-standby for [High Availability](high-availability.md#workflow-controller).

## Vertically Scaling

You can scale the controller vertically:

- If you have many workflows, increase `--workflow-workers` and `--workflow-ttl-workers`.
- Increase both `--qps` and `--burst`.

You will need to increase the controller's memory and CPU.

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
