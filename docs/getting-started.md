# Getting Started

To see how Argo works, you can run examples of simple workflows and workflows that use artifacts.
For the latter, you'll set up an artifact repository for storing the artifacts that are passed in
the workflows. Here are the requirements and steps to run the workflows.

## 0. Requirements
* Kubernetes 1.9 or later
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* Have a [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) file (default location is `~/.kube/config`)
* [Helm](https://helm.sh/docs/intro/quickstart/) (Optional): this tutorial uses helm to install MinIO, if you are going to use it (to try out "artifact-passing" for example).

## 1. Download the Argo CLI

Download the latest Argo CLI from our [releases page](https://github.com/argoproj/argo/releases).

## 2. Install the Controller

```sh
kubectl create namespace argo
kubectl apply -n argo -f https://raw.githubusercontent.com/argoproj/argo/stable/manifests/install.yaml
```

Namespaced installs as well as installs with MinIO and/or a database built in [are also available](https://github.com/argoproj/argo/tree/stable/manifests).

Examples below will assume you've installed argo in the `argo` namespace. If you have not, adjust
the commands accordingly.

NOTE: On GKE, you may need to grant your account the ability to create new `clusterrole`s

```sh
kubectl create clusterrolebinding YOURNAME-cluster-admin-binding --clusterrole=cluster-admin --user=YOUREMAIL@gmail.com
```

## 3. Configure the service account to run Workflows

### Roles, RoleBindings, and ServiceAccounts

In order for Argo to support features such as artifacts, outputs, access to secrets, etc. it needs to communicate with Kubernetes resources
using the Kubernetes API. To communicate with the Kubernetes API, Argo uses a `ServiceAccount` to authenticate itself to the Kubernetes API.
You can specify which `Role` (i.e. which permissions) the `ServiceAccount` that Argo uses by binding a `Role` to a `ServiceAccount` using a `RoleBinding`

Then, when submitting Workflows you can specify which `ServiceAccount` Argo uses using:

```sh
argo submit --serviceaccount <name>
```

When no `ServiceAccount` is provided, Argo will use the `default` `ServiceAccount` from the namespace from which it is run, which will almost always have insufficient privileges by default.

For more information about granting Argo the necessary permissions for your use case see [Workflow RBAC](workflow-rbac.md).

### Granting admin privileges

For the purposes of this demo, we will grant the `default` `ServiceAccount` admin privileges (i.e., we will bind the `admin` `Role` to the `default` `ServiceAccount` of the current namespace):

```sh
kubectl create rolebinding default-admin --clusterrole=admin --serviceaccount=argo:default -n argo
```

**Note that this will grant admin privileges to the `default` `ServiceAccount` in the namespace that the command is run from, so you will only be able to
run Workflows in the namespace where the `RoleBinding` was made.**

## 4. Run Sample Workflows
```sh
argo submit -n argo --watch https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
argo submit -n argo --watch https://raw.githubusercontent.com/argoproj/argo/master/examples/coinflip.yaml
argo submit -n argo --watch https://raw.githubusercontent.com/argoproj/argo/master/examples/loops-maps.yaml
argo list -n argo
argo get xxx-workflow-name-xxx -n argo
argo logs xxx-pod-name-xxx -n argo #from get command above
```

Additional examples and more information about the CLI are available on the [Argo Workflows by Example](../examples/README.md) page.

You can also create Workflows directly with `kubectl`. However, the Argo CLI offers extra features
that `kubectl` does not, such as YAML validation, workflow visualization, parameter passing, retries
and resubmits, suspend and resume, and more.
```sh
kubectl create -n argo -f https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
kubectl get wf -n argo
kubectl get wf hello-world-xxx -n argo
kubectl get po -n argo --selector=workflows.argoproj.io/workflow=hello-world-xxx
kubectl logs hello-world-yyy -c main -n argo
```


## 5. Install an Artifact Repository

Argo supports S3 (AWS, GCS, Minio) and Artifactory as artifact repositories. Instructions on how to configure artifact repositories are available on the [Configuring your Artifact Repository](configure-artifact-repository.md) page.

This tutorial uses Minio for the sake of portability.

Install Minio:
```sh
helm install argo-artifacts stable/minio \
  -n argo \
  --set service.type=LoadBalancer \
  --set defaultBucket.enabled=true \
  --set defaultBucket.name=my-bucket \
  --set persistence.enabled=false \
  --set fullnameOverride=argo-artifacts
```

Assuming Minio was installed into the "argo" namespace, the following `kubectl` command will expose the Minio UI. Login to the Minio UI using a web browser (port 9000).
```sh
kubectl get service argo-artifacts -o wide -n argo
```
On Minikube:
```sh
minikube service --url argo-artifacts -n argo
```

NOTE: When minio is installed via Helm, it uses the following hard-wired default credentials,
which you will use to login to the UI:
* AccessKey: `AKIAIOSFODNN7EXAMPLE`
* SecretKey: `wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`

There should be a bucket named `my-bucket`. If not create one from the Minio UI.

## 6. Reconfigure the workflow controller to use the Minio artifact repository

Edit the `workflow-controller` `ConfigMap` to reference the service name (`argo-artifacts`) and
secret (`argo-artifacts`) created by the Helm install:

Edit the `workflow-controller` `ConfigMap`:
```sh
kubectl edit cm -n argo workflow-controller-configmap
```
Add the following:
```yaml
data:
  artifactRepository: |
    s3:
      bucket: my-bucket
      endpoint: argo-artifacts:9000
      insecure: true
      # accessKeySecret and secretKeySecret are secret selectors.
      # It references the k8s secret named 'argo-artifacts'
      # which was created during the minio helm install. The keys,
      # 'accesskey' and 'secretkey', inside that secret are where the
      # actual minio credentials are stored.
      accessKeySecret:
        name: argo-artifacts
        key: accesskey
      secretKeySecret:
        name: argo-artifacts
        key: secretkey
```

NOTE: the Minio secret is retrieved from the namespace you use to run Workflows. If Minio is
installed in a different namespace then you will need to create a copy of its secret in the
namespace you use for Workflows.

## 7. Run a workflow which uses artifacts
```sh
argo submit -n argo https://raw.githubusercontent.com/argoproj/argo/master/examples/artifact-passing.yaml
```

## 8. Access the Argo UI

> v2.5 and after

```
kubectl -n argo port-forward deployment/argo-server 2746:2746
```

Then visit: http://127.0.0.1:2746

See the [Argo Server documentation](./argo-server.md) for config options, authentication,
managed namespaces, etc.

> v2.4 and before

By default, the Argo UI service is not exposed with an external IP. To access the UI, use one of the
following:

### Method 1: kubectl port-forward
```
kubectl -n argo port-forward deployment/argo-ui 8001:8001
```
Then visit: http://127.0.0.1:8001

### Method 2: kubectl proxy
```
kubectl proxy
```
Then visit: http://127.0.0.1:8001/api/v1/namespaces/argo/services/argo-ui/proxy/

NOTE: artifact download and webconsole is not supported using this method

### Method 3: Expose a LoadBalancer
Update the argo-ui service to be of type `LoadBalancer`.
```
kubectl patch svc argo-ui -n argo -p '{"spec": {"type": "LoadBalancer"}}'
```
Then wait for the external IP to be made available:
```
kubectl get svc argo-ui -n argo
NAME      TYPE           CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
argo-ui   LoadBalancer   10.19.255.205   35.197.49.167   80:30999/TCP   1m
```

NOTE: On Minikube, you won't get an external IP after updating the service -- it will always show
`pending`. Run the following command to determine the Argo UI URL:
```
minikube service -n argo --url argo-ui
```
