# Multi-Cluster Workflows

Argo Workflows v3.4 will introduce a feature to allow you to run workflows where script, resource, and container
templates can be run in a different cluster or namespace to the workflow itself:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: main-
spec:
  entrypoint: main
  templates:
    - name: main
      cluster: cluster-1
      namespace: default
      container:
        image: argoproj/argosay:v2
```

I'm going to make some assumptions:

* Your default Kubernetes context is the primary cluster.
* There is a Kubernetes context for the remote cluster (named `cluster-1`).

Update the config map with permissions:

```yaml
apiVersion: v1
data:
  model.conf: |-
    [request_definition]
    r = sub, obj

    [policy_definition]
    p = sub, obj

    [policy_effect]
    e = some(where (p.eft == allow))

    [matchers]
    m = r.sub == r.obj || keyMatch(r.sub, p.sub) && keyMatch(r.obj, p.obj)
  policy.csv: |
    # Workflows in the "argo" namespace may create resources in the "cluster-1" cluster's "default" namespace
    p, cluster-0:argo, cluster-1:default

    # Workflows in the "argo" namespace may create resources in the primary cluster's "default" namespace
    p, cluster-0:argo, cluster-0:default
kind: ConfigMap
metadata:
  name: workflow-controller-authz
```

The workflow controller must be configured with the name of its cluster:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-controller
spec:
  template:
    spec:
      containers:
        - name: workflow-controller
          env:
            - name: ARGO_CLUSTER
              value: cluster-0
```

Set-up resources in the remote and primary cluser:

```bash
# install resources into remote cluster
kubectl kustomize --load-restrictor=LoadRestrictionsNone manifests/quick-start/cluster-1 | kubectl --context=cluster-1 -n default apply -f -

# install profile into primary cluster
argo cluster get-profile cluster-1 default argo.cluster-0 argo --server=https://`ipconfig getifaddr en0`:`kubectl config view --raw --minify --context=cluster-1|grep server|cut -c 29-` --insecure-skip-tls-verify | kubectl -n argo apply -f  -

# create default bindings for the executor
kubectl --context=cluster-1 create role executor --verb=create,patch --resource=workflowtaskresults.argoproj.io
kubectl --context=cluster-1 create rolebinding default-executor --role=executor --user=system:serviceaccount:default:default
```

Restart the controller:

```bash
kubectl rollout restart deploy/workflow-controller
```

Finally, run a test workflow.

### Limitations

* Only resources can be created in the other cluster. Resources that are automatically created (such as artifact
  repositories, persistent volume claims, pod disruption budgets) are not currently supported.
* In the API and UI, only logs for resources created in the workflow's namespace are currently supported.

### Scaling

Workflow controllers running multi-cluster workflows will open additional connections for each cluster.

### Pod Garbage Collection

If a pod is created in another cluster, and the parent workflow is deleted, then Argo must garbage collect it. Normally,
Kubernetes would do this.

⚠️ This garbage collection is done on best effort, and that might be long time after the workflow is deleted. To
mitigate this, use `podGCStrategy`.

## Authorization

We use Casbin for access control. The default model.conf should work for most cases. When you have mulitple cluster, you
need to write your own policy.csv to replace the default one.

```csv
# Workflows in the "argo" namespace may create resources in the "cluster-1" cluster's "default" namespace
p, cluster-0:argo, cluster-1:default

# Workflows in the "argo" namespace may create resources in the primary cluster's "default" namespace
p, cluster-0:argo, cluster-0:default
```

Running a workflow in its own namespace is allowed by the model.conf.

## Scaling

An controller configured with multiple clusters will need to be scaled-up to match load.

## Reliability

Network requests will be disrupted by network problems more often.
