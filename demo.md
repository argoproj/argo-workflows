# Argo v2.0 Demo

## Requirements:
* Kubernetes 1.8
* A working `kubectl` and ~/.kube/config

## 1. Download argo

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

## 2. Install the controller
```
$ argo install
```

If the cluster has legacy authentication disabled, create
a service account and role binding with admin privileges.
Then specify the service account during install. 

```
$ kubectl create serviceaccount --namespace kube-system argo
$ kubectl create clusterrolebinding argo-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:argo
$ argo install --service-account argo
```

## 3. Run some examples workflows
```
$ argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
$ argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/coinflip.yaml
$ argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/loops-maps.yaml
$ argo list
$ argo get xxx-workflow-name-xxx
$ argo logs xxx-pod-name-xxx #from get command above
```

You can also run workflows directly with kubectl. Usually, the argo CLI is preferred because it validates your yaml, presents more human friendly output, and requires less typing.
```
$ kubectl create -f https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
$ kubectl get wf
$ kubectl get wf hello-world-xxx
$ kubectl get po --selector=workflows.argoproj.io/workflow=hello-world-xxx --show-all
$ kubectl logs hello-world-yyy -c main
```

## 4. Install an artifact repository
```
$ brew install kubernetes-helm #mac
$ helm init
$ helm install stable/minio --name argo-artifacts
```

Login to minio and create a bucket (my-bucket). When minio is installed via Helm, it uses the following hard-wired default credentials:
* AccessKey: AKIAIOSFODNN7EXAMPLE
* SecretKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

## 5. Reconfigure the workflow controller to use the minio artifact repository configured in step 4.
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
