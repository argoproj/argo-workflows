# Argo v2.0 Demo

To see how Argo works, you can run examples of simple workflows and workflows that use artifacts. For the latter, you'll set up an artifact repository for storing the artifacts that are passed in the workflows. Here are the requirements and steps to run the workflows.

## Requirements
* Installed Kubernetes 1.8 or later
* Installed the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool
* Have a [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) file (default location is `~/.kube/config`).

## 1. Download Argo

On Mac:
```
$ curl -sSL -o ./argo https://github.com/argoproj/argo/releases/download/v2.0.0-alpha2/argo-darwin-amd64
$ chmod +x argo 
```
On Linux:
```
$ curl -sSL -o ./argo https://github.com/argoproj/argo/releases/download/v2.0.0-alpha2/argo-linux-amd64
$ chmod +x argo 
```

## 2. Install the Controller and UI
```
$ argo install
```
Installation command does not configure access for argo UI. Please use following command to create externally accessable Kubernetes service:

```
$ kubectl create -f https://raw.githubusercontent.com/argoproj/argo/master/ui/deploy/service.yaml --namespace kube-system
```

Service namespace should correspond to namespace chosen during argo installation (kube-system is default namespace).

## 3. Run Simple Example Workflows
```
$ argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
$ argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/coinflip.yaml
$ argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/loops-maps.yaml
$ argo list
$ argo get xxx-workflow-name-xxx
$ argo logs xxx-pod-name-xxx #from get command above
```

You can also run workflows directly with kubectl. However, the Argo CLI offers extra features that kubectl does not, such as YAML validation, workflow visualization, and overall less typing.
```
$ kubectl create -f https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
$ kubectl get wf
$ kubectl get wf hello-world-xxx
$ kubectl get po --selector=workflows.argoproj.io/workflow=hello-world-xxx --show-all
$ kubectl logs hello-world-yyy -c main
```

## 4. Install an Artifact Repository

You'll create the artifact repo using Minio.
```
$ brew install kubernetes-helm #mac
$ helm init
$ helm install stable/minio --name argo-artifacts
```
## 5. Login to Minio and create a bucket

NOTE: When Minio is installed via Helm, it uses the following hard-wired default credentials:
* AccessKey: AKIAIOSFODNN7EXAMPLE
* SecretKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

## 5. Reconfigure the workflow controller to use the Minio artifact repository configured in step 4.
Look at minio created resources:
```
# kubectl get all -l release=argo-artifacts
```
Edit the workflow-controller config to reference the service name (argo-artifacts-minio-svc) and secret (argo-artifacts-minio-user) created by the helm install:
```
$ kubectl edit configmap workflow-controller-configmap -n kube-system
...
    executorImage: argoproj/argoexec:v2.0.0-alpha2
    artifactRepository:
      s3:
        bucket: my-bucket
        endpoint: argo-artifacts-minio-svc.default:9000
        insecure: true
        accessKeySecret:
          name: argo-artifacts-minio-user
          key: accesskey
        secretKeySecret:
          name: argo-artifacts-minio-user
          key: secretkey
```

## 6. Run a workflow which uses artifacts
```
$ argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/artifact-passing.yaml
```
