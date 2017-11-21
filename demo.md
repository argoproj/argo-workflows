# Argo v2.0 Demo

To see how Argo works, you can run examples of simple workflows and workflows that use artifacts. For the latter, you'll set up an artifact repository for storing the artifacts that are passed in the workflows. Here are the requirements and steps to run the workflows.

## Requirements
* Installed Kubernetes 1.8 or later
* Installed the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool
* Have a [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) file (default location is `~/.kube/config`).

## 1. Download Argo

On Mac:
```
$ curl -sSL -o ./argo https://github.com/argoproj/argo/releases/download/v2.0.0-alpha1/argo-darwin-amd64
$ chmod +x argo 
```
On Linux:
```
$ curl -sSL -o ./argo https://github.com/argoproj/argo/releases/download/v2.0.0-alpha1/argo-linux-amd64
$ chmod +x argo 
```

## 2. Install the Controller
```
$ argo install
```

 NOTE: If your Kubernetes cluster has legacy authentication disabled, you must create a service account and a [cluster role binding](https://kubernetes.io/docs/admin/authorization/rbac/#kubectl-create-clusterrolebinding) with admin privileges. Then when you install Argo, specify the newly created service account. 

```
$ kubectl create serviceaccount --namespace kube-system argo
$ kubectl create clusterrolebinding argo-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:argo
$ argo install --service-account argo
```
 Where:
 
 * `kube-system argo` is the namespace that the `argo` serviceaccount has access to.
 * `argo-cluster-rule` is the name for the cluster role binding
 
 
## 3. Run Simple Example Workflows
```
$ argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
$ argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/coinflip.yaml
$ argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/loops-maps.yaml
$ argo list
$ argo get xxx-workflow-name-xxx
$ argo logs xxx-pod-name-xxx #from get command above
```

You can also run workflows directly with kubectl. However, the Argo CLI offers extra features that kubectl does not, such as the Argo CLI validates your YAML, displays a more human-friendly output, and requires less typing.
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
    executorImage: argoproj/argoexec:latest
    artifactRepository:
      s3:
        bucket: my-bucket
        endpoint: argo-artifacts-minio-svc:9000
        insecure: true
        accessKeySecret:
          name: argo-artifacts-minio-user
          key: accesskey
        secretKeySecret:
          name: argo-artifacts-minio-user
          key: secretkey
```

Restart the workflow-controller pod for the config to take effect
```
$ kubectl get pods -n kube-system -l app=workflow-controller
$ kubectl delete pod <workflow-controller-podname> -n kube-system
```

## 6. Run a workflow which uses artifacts
```
$ argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/artifact-passing.yaml
```
