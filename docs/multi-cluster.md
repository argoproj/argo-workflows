# Multi-Cluster

You can run workflows where one or more tasks are run in another cluster or namespace.

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

Create a secret named `kubeconfig` that has a single item `value` with that value:

```bash
kubectl -n argo create secret generic kubeconfig --from-file=value=cluster-1-kubeconfig.yaml
```

Restart your workflow controller and make sure you see the new cluster printed in your log, and no errors.

```
controller | time="2021-08-27T14:10:56.273Z" level=info msg=cluster cluster=cluster-1 managedNamespace=remote
controller | time="2021-08-27T14:10:56.273Z" level=info msg=cluster cluster=@in-cluster managedNamespace=local
```

(2) Create any secrets (e.g. to archive logs or artifacts) you need, for example:

```bash
kubectl -n remote apply -f https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start/base/minio/my-minio-cred-secret.yaml
```

(3) If your workflow uses a non-default service account, create that:

```bash
kubectl -n remote apply -f https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start/base/workflow-role.yaml
kubectl -n remote create sa workflow
kubectl -n remote create rolebinding workflow --role=workflow-role --serviceaccount=remote:workflow
```

(4) In the **remote cluster**, create a service account with permission to create pods in the namespace you want to run
pods in. For example:

```
kubectl -n remote create role remote --verb=create --resource=pods 
kubectl -n remote create sa remote
kubectl -n remote create rolebinding remote --role=remote --serviceaccount=default:remote
```

This will create a service account token in the remote namespace:

```bash
SECRET=$(kubectl -n default get sa remote -o=jsonpath='{.secrets[0].name}')
TOKEN=$(kubectl get -n default secret $SECRET -o=jsonpath='{.data.token}' | base64 --decode)
```

(5) Create a kubeconfig secret in the namespace where workflows will be created. This only needs to contain users and
context.

There must be a context that:

* Has same cluster name as the remote cluster.
* Has same namespace as the remote cluster, or the namespace in omitted.

```bash
sed "s/TOKEN/$TOKEN/" > local-kubeconfig.yaml <<END
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
kubectl -n local create secret generic kubeconfig --from-file=value=local-kubeconfig.yaml
```

(6) Finally, create a multi-cluster workflow:

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
      namespace: default
      container:
        image: docker/whalesay
```
