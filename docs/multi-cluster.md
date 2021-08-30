# Multi-Cluster

![alpha](assets/alpha.svg)

> Since v3.3

You can run workflows where one or more step are run in another cluster or namespace. This is done by specifying the the
cluster and namespace in your template:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: multi-cluster
  namespace: local
spec:
  entrypoint: main
  serviceAccountName: workflow
  templates:
    - name: main
      cluster: cluster-1
      namespace: remote
      container:
        image: docker/whalesay
```

Clusters are configured in the `argo` namespace in a secret named `argo-kubeconfig` which has one context per cluster,
allowing the controller to both watch for completed pods, and delete pods once the workflow is completed.

To prevent workflows from being deleted before pods have been delete in other clusters and namespaces, we add a
finalizer to them.

Prior to multi-cluster support, the security model meant that pods would only be created in the workflow's namespace.
With multi-cluster, users may create them in any namespace, so long as they have a kube config within their namespace
that allows them to. That must have a user that is allowed by the remote cluster to create pods in the remote namespace.
That should be a different user to the cluster's kube config in the `argo` namespace. This allows the `argo` namespace
to have different permissions to the user namespace. The controller kube config does not need to create pods,
only `list/watch/delete`, unlike the user kube config only needs `create pod` within specific namespace.

## Scaling

The controllers create one Kubernetes API connection per context within kube config:

1. If only a few remote namespaces need watching, then create one context entry per namespace.
1. If multiple namespaces within a cluster need watching, then create one context entry per cluster with cluster role
   and role-binding.

## Known Issues

* Logs cannot be accessed from the user interface.
* Cluster and namespace set on DAG and steps templates are ignored; they are not inherited by container or script
  templates.

## Set-up

You may need to use a binary to authenticate with your Kubernetes provider. You can find out by looking at you kube
config `kubectl config view` to see if there's an exec section:

```yaml
users:
  - name: arn:aws:eks:...
    user:
      exec:
        apiVersion: client.authentication.k8s.io/v1alpha1
        args:
          - ...
        command: aws
```

You could use an init container to get the binary, and then mount on main
containers ([example](https://github.com/argoproj/argo-workflows/issues/3523#issuecomment-907857660)). In production,
you should build your own container image which you can then scan for vulnerabilities, something like:

```Dockerfile
FROM ubuntu as get-binary

# get the binary
# make the binary executable 

FROM quay.io/argoproj/workflow-controller:v0.0.0-dev-mc-0 as workflow-controller

COPY --chown=8737 --from=get-binary /aws /aws
```

## Configuration

To make things clear, in the examples we've:

1. Installed argo in the `argo` namespace.
2. Will be creating our workflows in the `local` namespace.
3. Be creating pods in a cluster named `cluster-1` in the `remote` namespace.

In Argo's **system namespace**:

(1) Create a secret container the kube config for you clusters.

Firstly, create a KUBECONFIG file that only contains the clusters and users you need.

The file should have exactly one context per cluster, and the context's name must be the same as the cluster's name.

```yaml
apiVersion: v1
clusters:
  - cluster:
      certificate-authority-data: xxx
      server: https://1.2.3.4:6443
    name: cluster-1
contexts:
  - context:
      cluster: cluster-1
      namespace: remote
      user: cluster-1
    name: cluster-1
kind: Config
preferences: { }
users:
  - name: cluster-1
    user:
      token: xxx
```

If you want to allow creation of pods in multiple namespaces, then you must leave `namespace` empty.

Create a secret named `argo-kubeconfig` that has a single item `value` with that value:

```bash
kubectl -n argo create secret generic argo-kubeconfig --from-file=value=argo-kubeconfig.yaml
```

(2) Updated `configaps/workflow-controller-configmap` to add a unique name for your local cluster:

```bash
kubectl patch cm workflow-controller-configmap -p '{"data":{"cluster":"main"}}'
```

That must be unique within all clusters running workflows. In practice that usually means unique within your
organisation.

ℹYou might be using the old config map format. In which case, you'll need to edit the key `config`.

Restart your workflow controller and make sure you see the new cluster printed in your log, and no errors.

```
controller | time="2021-08-27T14:10:56.273Z" level=info msg=cluster cluster=cluster-1 managedNamespace=remote
controller | time="2021-08-27T14:10:56.273Z" level=info msg=cluster cluster=@in-cluster managedNamespace=local
```

(3) Create any secrets (e.g. to archive logs or artifacts) you need, for example:

```bash
kubectl -n remote apply -f https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start/base/minio/my-minio-cred-secret.yaml
```

(4) If your workflow uses a non-default service account, create that:

```bash
kubectl -n remote apply -f https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start/base/workflow-role.yaml
kubectl -n remote create sa workflow
kubectl -n remote create rolebinding workflow --role=workflow-role --serviceaccount=remote:workflow
```

(5) In the **remote cluster**, create a service account with permission to create pods in the namespace you want to run
pods in. For example:

```
kubectl -n remote create role remote --verb=create --resource=pods 
kubectl -n remote create sa remote
kubectl -n remote create rolebinding remote --role=remote --serviceaccount=remote:remote
```

This will create a service account token in the remote namespace:

```bash
SECRET=$(kubectl -n default get sa remote -o=jsonpath='{.secrets[0].name}')
TOKEN=$(kubectl get -n default secret $SECRET -o=jsonpath='{.data.token}' | base64 --decode)
```

(6) Create a kube config secret in the namespace where workflows will be created. This only needs to contain users and
context. It must be named `${serviceAccountName}-kubeconfig` (in these examples the service account name is "workflow"): 

There must be a context that:

* Has same cluster name as the remote cluster.
* Has same namespace as the remote cluster, or the namespace in omitted.

```bash
sed "s/TOKEN/$TOKEN/" > workflow-kubeconfig.yaml <<END
apiVersion: v1
contexts:
  - context:
      cluster: cluster-1
      namespace: remote
      user: cluster-1
    name: cluster-1
current-context: cluster-1
kind: Config
preferences: { }
users:
  - name: cluster-1
    user:
      token: TOKEN
END
kubectl -n local create secret generic workflow-kubeconfig --from-file=value=workflow-kubeconfig.yaml
```

(7) Finally, create a multi-cluster workflow:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: multi-cluster
  namespace: local
spec:
  entrypoint: main
  serviceAccountName: workflow
  templates:
    - name: main
      cluster: cluster-1
      namespace: remote
      container:
        image: docker/whalesay
```
