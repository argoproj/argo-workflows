# Scaling

For running large workflows, you'll typically need to scale the controller to match.

## Horizontally Scaling

You cannot horizontally scale the controller.

## Vertically Scaling

You can scale the controller vertically:

- If you have workflows with many steps, increase `--pod-workers`.
- If you have many workflows, increase `--workflow-workers`. 
- Increase both `--qps` and `--burst`.

You will need to increase the controller's memory and CPU.

## Sharding

### One Install Per Namespace

Rather than running a single installation in your cluster, run one per namespace using the `--namespaced` flag.

### Instance ID

Within a cluster can use instance ID to run N Argo instances within a cluster. 

Create one namespace for each Argo, e.g. `argo-i1`, `argo-i2`:.

Edit [workflow-controller-configmap.yaml](workflow-controller-configmap.yaml) for each namespace to set an instance ID.

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
    instanceID: i1
```

> v2.9 and after

You'll may need to pass the instance ID to the CLI:

```
argo --instance-id i1 submit my-wf.yaml
```

You do not need to have one instance ID per namespace, you could have many or few.
