# Argo Getting Started

To see how Argo works, you can run examples of simple workflows and workflows that use artifacts.
For the latter, you'll set up an artifact repository for storing the artifacts that are passed in
the workflows. Here are the requirements and steps to run the workflows.

## Requirements
* Installed Kubernetes 1.9 or later
* Installed [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* Have a [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) file (default location is `~/.kube/config`).

## 1. Download Argo

Download the latest Argo binary version from https://github.com/argoproj/argo/releases/latest.

Also available in Mac Homebrew:
```
brew install argoproj/tap/argo
```

Also you can use this command to install for Linux 

```
curl -sSL -o /usr/local/bin/argo https://github.com/argoproj/argo/releases/download/v2.3.0/argo-linux-amd64
chmod +x /usr/local/bin/argo
```

## 2. Install the Controller and UI
```
kubectl create namespace argo
kubectl apply -n argo -f https://raw.githubusercontent.com/argoproj/argo/stable/manifests/install.yaml
```
NOTE: On GKE, you may need to grant your account the ability to create new clusterroles
```
kubectl create clusterrolebinding YOURNAME-cluster-admin-binding --clusterrole=cluster-admin --user=YOUREMAIL@gmail.com
```

## 3. Configure the service account to run workflows

To run all of the examples in this guide, the 'default' service account is too limited to support
features such as artifacts, outputs, access to secrets, etc... For demo purposes, run the following
command to grant admin privileges to the 'default' service account in the namespace 'default':
```
kubectl create rolebinding default-admin --clusterrole=admin --serviceaccount=default:default
```
For the bare minimum set of privileges which a workflow needs to function, see
[Workflow RBAC](docs/workflow-rbac.md). You can also submit workflows which run with a different
service account using:
```
argo submit --serviceaccount <name>
```

## 4. Run Simple Example Workflows
```
argo submit --watch https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
argo submit --watch https://raw.githubusercontent.com/argoproj/argo/master/examples/coinflip.yaml
argo submit --watch https://raw.githubusercontent.com/argoproj/argo/master/examples/loops-maps.yaml
argo list
argo get xxx-workflow-name-xxx
argo logs xxx-pod-name-xxx #from get command above
```

You can also create workflows directly with kubectl. However, the Argo CLI offers extra features
that kubectl does not, such as YAML validation, workflow visualization, parameter passing, retries
and resubmits, suspend and resume, and more.
```
kubectl create -f https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
kubectl get wf
kubectl get wf hello-world-xxx
kubectl get po --selector=workflows.argoproj.io/workflow=hello-world-xxx --show-all
kubectl logs hello-world-yyy -c main
```

Additional examples are available [here](https://github.com/argoproj/argo/blob/master/examples/README.md).

## 5. Install an Artifact Repository

Argo supports S3 (AWS, GCS, Minio) as well as Artifactory as artifact repositories. This tutorial
uses Minio for the sake of portability. Instructions on how to configure other artifact repositories
are [here](https://github.com/argoproj/argo/blob/master/ARTIFACT_REPO.md).
```
helm install stable/minio \
  --name argo-artifacts \
  --set service.type=LoadBalancer \
  --set defaultBucket.enabled=true \
  --set defaultBucket.name=my-bucket \
  --set persistence.enabled=false \
  --set fullnameOverride=argo-artifacts
```

Login to the Minio UI using a web browser (port 9000) after exposing obtaining the external IP using `kubectl`.
```
kubectl get service argo-artifacts-mini -o wide
```
On Minikube:
```
minikube service --url argo-artifacts-mini
```

NOTE: When minio is installed via Helm, it uses the following hard-wired default credentials,
which you will use to login to the UI:
* AccessKey: AKIAIOSFODNN7EXAMPLE
* SecretKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

Create a bucket named `my-bucket` from the Minio UI.

## 6. Reconfigure the workflow controller to use the Minio artifact repository

Edit the workflow-controller config map to reference the service name (argo-artifacts) and
secret (argo-artifacts) created by the helm install:
```
kubectl edit cm -n argo workflow-controller-configmap
...
data:
  config: |
    artifactRepository:
      s3:
        bucket: my-bucket
        endpoint: argo-artifacts.default:9000
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

NOTE: the Minio secret is retrieved from the namespace you use to run workflows. If Minio is
installed in a different namespace then you will need to create a copy of its secret in the
namespace you use for workflows.

## 7. Run a workflow which uses artifacts
```
argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/artifact-passing.yaml
```

## 8. Access the Argo UI

By default, the Argo UI service is not exposed with an external IP. To access the UI, use one of the
following methods:

#### Method 1: kubectl port-forward
```
kubectl -n argo port-forward deployment/argo-ui 8001:8001
```
Then visit: http://127.0.0.1:8001

#### Method 2: kubectl proxy
```
kubectl proxy
```
Then visit: http://127.0.0.1:8001/api/v1/namespaces/argo/services/argo-ui/proxy/

NOTE: artifact download and webconsole is not supported using this method

#### Method 3: Expose a LoadBalancer
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
