# Multi-cluster

Argo allows you to run workflows where the workflow's tasks are run in a different cluster and namespace to the
workflow.

## Terminology

When running workflows that creates resources (i.e. run tasks/steps) in other clusters and namespaces.

* The **local cluster** is where you'll create your workflows in. All cluster must be given a unique name, so in the
  examples we'll call this `cluster-0`.
* The **workflow namespace** is where workflow is, which may be different to the resource's namespace. In the
  examples, `argo`.
* The **remote cluster** is where the workflow may create pods. In the examples, `cluster-1`.
* The **remote namespace** is where remote resources are created. In the examples, `default`.

## Configuration

I'm going to make some assumptions:

* Your default Kubernetes context is the local cluster.
* There is aKubernetes context for the remote cluster name `cluster-1`.

You must install the `workflowtastresults` CRD in the remote cluster:

```bash
kubectl --context=cluster-1 apply -f manifests/base/crds/minimal/argoproj.io_workflowtaskresults.yaml
```

The service account used by your workflow pods in the remote cluster will need the standard permissions. If you use
the `default` service account, then this will work:

```bash
kubectl --context=cluster-1 create role executor --verb=create,patch --resource=workflowtaskresults.argoproj.io
kubectl --context=cluster-1 create role default-executor --role=executor --user=system:serviceaccount:default:default
```

We recommend you create service account the remote cluster for the local cluster to use:

<!-- this block of code is replicated in Makefile, if you change it here, copy it there -->

```bash
kubectl --context=cluster-1 create serviceaccount argo-cluster-0
kubectl --context=cluster-1 create clusterrole pod-reconciller --verb=create,patch,delete,list,watch --resource=pods,pods/exec
kubectl --context=cluster-1 create clusterrole workflowtaskresult-reconciller --verb=list,watch,deletecollection --resource=workflowtaskresults.argoproj.io
kubectl --context=cluster-1 create clusterrolebinding argo-cluster-0-pod-reconciller --clusterrole=pod-reconciller --user=system:serviceaccount:default:argo-cluster-0
kubectl --context=cluster-1 create clusterrolebinding argo-cluster-0-workflowtaskresult-reconciller --clusterrole=workflowtaskresult-reconciller --user=system:serviceaccount:default:argo-cluster-0
```

In this example, I've used `argo-cluster-0` to indicate that the service account belongs to Argo running in the local
cluster, which I named `cluster-0`.

Argo can only manage pods in clusters in can connect to - ones it has a `kubeconfig` for. These are called **profiles**.
A profile is a secret labelled with `workflows.argoproj.io/cluster`.

Create a profile. You can use `./hack/print-kubeconfig.sh` to get the `kubeconfig`:

```bash
kubectl create secret generic cluster-1 --from-literal="kubeconfig=`./hack/print-kubeconfig.sh cluster-1 default argo-cluster-0`"
kubectl label secret cluster-1 workflows.argoproj.io/cluster=cluster-1
```

By default, Argo will try and use the profile to read and write resources in all namespaces of the remote cluster, you
probably don't want that. Annotate the secret like this:

```bash
kubectl annotate secret cluster-1 workflows.argoproj.io/namespace=default
```

This will only be used for workflows in the same namespace as the secret, if you want a specific namespace:

```bash
kubectl annotate secret cluster-1 workflows.argoproj.io/workflow-namespace=default
```

Or for any namespace:

```bash
kubectl annotate secret cluster-1 workflows.argoproj.io/workflow-namespace=
```

You need need a `kubeconfig` secret for each action:

* `read` - one that has `pod list,watch` permissions, used to determine the status of the pod. This must be created in
  the system namespace, typically `argo`.
* `write` - one that has `pod create,patch,delete` permissions, used to create the pod.

You may also create a single secret for both read and write.

The workflow controller must be configured with it's name:

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

Finally, you can run a test workflow:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: multi-cluster-
spec:
  entrypoint: main
  artifactRepositoryRef:
    key: empty
  templates:
    - name: main
      cluster: cluster-1
      namespace: default
      container:
        image: argoproj/argosay:v2
```

## Limitations

* Only resources can be created in the other cluster. Resources that are automatically created (such as artifact
  repositories, persistent volume claims, pod disruption budgets) are not currently supported.
* In the API and UI, only logs for resources created in the workflow's namespace are currently supported.

## Scaling

Workflow controllers running multi-cluster workflows will open additional connections for each cluster.

## Labels

It is worthwhile understand how Argo uses labels. Some facts:

* It is not possible to create an ownership reference between resources in different namespaces or clusters.
* It is possible for two different Argos to create pods in the same namespace that belong to different workflows.

So this creates problems:

* How do I make sure pods are deleted if the workflow is deleted?
* How do I know which pod belongs to which workflow?

This is solved using labels:

* `workflows.argoproj.io/cluster` tells you which the cluster of the workflow.
* `workflows.argoproj.io/workflow-namespace` tells you which the namespace of the workflows.

These labels are only applied if an ownership reference cannot be created, i.e. if if the pod is created in different
cluster or namespace to the workflow.

## Pod Garbage Collection

If a pod is created in another cluster, and the parent workflow is deleted, then Argo must garbage collect it. Normally,
Kubernetes would do this.

⚠️ This garbage collection is done on best effort, and that might be long time after the workflow is deleted. To
mitigate this, use `podGCStrategy`.

