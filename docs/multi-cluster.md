# Multi-cluster

⚠️ Work in progress.

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

## Core Concepts

When running workflows that creates resources (i.e. run tasks/steps) in other clusters and namespaces.

* The **local cluster** is where you'll create your workflows in. All cluster must be given a unique name. In examples
  we'll call this `cluster-0`.
* The **workflow namespace** is where workflow is, which may be different to the resource's namespace. In the
  examples, `argo`.
* The **remote cluster** is where the workflow may create pods. In the examples, `cluster-1`.
* The **remote namespace** is where remote resources are created. In the examples, `default`.
* A **profile** is a configuration profile used to connect to a remote cluster.

## Configuration

I'm going to make some assumptions:

* Your default Kubernetes context is the local cluster.
* There is a Kubernetes context for the remote cluster (named `cluster-1`).

The workflow controller must be configured with the name of its cluster:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  # A unique name for the cluster.
  # It is acceptable for this to be a random UUID, but once set, it should not be changed.
  cluster: cluster-0
```

Restart the controller:

```bash
kubectl rollout restart deploy/workflow-controller
```

```bash
# install resources into remote cluster
kubectl kustomize --load-restrictor=LoadRestrictionsNone manifests/quick-start/cluster-1 | kubectl --context=cluster-1 -n default apply -f -

# install profile into local cluster
argo cluster get-profile cluster-1 default argo.cluster-0 --server=https://`ipconfig getifaddr en0`:`kubectl config view --raw --minify --context=cluster-1|grep server|cut -c 29-` --insecure-skip-tls-verify | kubectl -n argo apply -f  -
kubectl annotate secret argo.cluster-1 --overwrite workflows.argoproj.io/workflow-namespace=argo
kubectl annotate secret argo.cluster-1 --overwrite workflows.argoproj.io/namespace=default

# create default bindings for the executor
kubectl --context=cluster-1 create role executor --verb=create,patch --resource=workflowtaskresults.argoproj.io
kubectl --context=cluster-1 create rolebinding default-executor --role=executor --user=system:serviceaccount:default:default
```

Finally, run a test workflow.

## Limitations

* Only resources can be created in the other cluster. Resources that are automatically created (such as artifact
  repositories, persistent volume claims, pod disruption budgets) are not currently supported.
* In the API and UI, only logs for resources created in the workflow's namespace are currently supported.

## Scaling

Workflow controllers running multi-cluster workflows will open additional connections for each cluster.

## Pod Garbage Collection

If a pod is created in another cluster, and the parent workflow is deleted, then Argo must garbage collect it. Normally,
Kubernetes would do this.

⚠️ This garbage collection is done on best effort, and that might be long time after the workflow is deleted. To
mitigate this, use `podGCStrategy`.

