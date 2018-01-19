# Argo v2.0 Getting Started

To see how Argo works, you can run examples of simple workflows and workflows that use artifacts. For the latter, you'll set up an artifact repository for storing the artifacts that are passed in the workflows. Here are the requirements and steps to run the workflows.

## Requirements
* Installed Kubernetes 1.8 or later
* Installed the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool
* Have a [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) file (default location is `~/.kube/config`).

## 1. Download Argo

On Mac:
```
$ brew install argoproj/tap/argo
```
On Linux:
```
$ curl -sSL -o /usr/local/bin/argo https://github.com/argoproj/argo/releases/download/v2.0.0-beta1/argo-linux-amd64
$ chmod +x /usr/local/bin/argo
```

## 2. Install the Controller and UI
```
$ argo install
```
NOTE:
* On GKE with RBAC enabled, you may need to grant your account the ability to create new cluster roles
```
$ kubectl create clusterrolebinding YOURNAME-cluster-admin-binding --clusterrole=cluster-admin --user=YOUREMAIL@gmail.com
```
* The subsequent instructions below assume the installation of argo into the `kube-system` namespace (the default behavior). A different namespace can be chosen using the `argo install --install-namespace <name>` flag, in which case you should substitute `kube-system` with your chosen namespace in the examples below.

## 3. Configure the service account to run workflows (required for RBAC clusters)
For clusters with RBAC enabled, the 'default' service account is too limited to do any kind of meaningful work. Run the following command to grant admin privileges to the 'default' service account in the namespace 'default':
```
$ kubectl create rolebinding default-admin --clusterrole=admin --serviceaccount=default:default
```
NOTE: You can also submit workflows using a different service account using the `argo submit --serviceaccount <name>` flag.

## 4. Run Simple Example Workflows
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

Additional examples are availabe [here](https://github.com/argoproj/argo/blob/master/examples/README.md).

## 5. Install an Artifact Repository

Argo supports S3 (AWS, GCS, Minio) as well as Artifactory as artifact repositories. This tutorial uses Minio for the sake of portability. Instructions on how to configure other artifact repositories are [here](https://github.com/argoproj/argo/blob/master/ARTIFACT_REPO.md).
```
$ brew install kubernetes-helm # mac
$ helm init
$ helm install stable/minio --name argo-artifacts
```

Login to the Minio UI using a web browser (port 9000) after obtaining the external IP using `kubectl`.
```
$ kubectl get service argo-artifacts-minio-svc
```
On Minikube:
```
$ minikube service --url argo-artifacts-minio-svc
```

NOTE: When minio is installed via Helm, it uses the following hard-wired default credentials,
which you will use to login to the UI:
* AccessKey: AKIAIOSFODNN7EXAMPLE
* SecretKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

Create a bucket named `my-bucket` from the Minio UI.

## 6. Reconfigure the workflow controller to use the Minio artifact repository

Edit the workflow-controller config map to reference the service name (argo-artifacts-minio-svc) and secret (argo-artifacts-minio-user) created by the helm install:
```
$ kubectl edit configmap workflow-controller-configmap -n kube-system
...
    executorImage: argoproj/argoexec:v2.0.0-beta1
    artifactRepository:
      s3:
        bucket: my-bucket
        endpoint: argo-artifacts-minio-svc.default:9000
        insecure: true
        # accessKeySecret and secretKeySecret are secret selectors.
        # It references the k8s secret named 'argo-artifacts-minio-user'
        # which was created during the minio helm install. The keys,
        # 'accesskey' and 'secretkey', inside that secret are where the
        # actual minio credentials are stored.
        accessKeySecret:
          name: argo-artifacts-minio-user
          key: accesskey
        secretKeySecret:
          name: argo-artifacts-minio-user
          key: secretkey
```

The Minio secret is retrived from the namespace you use to run workflows. If Minio is installed in a different namespace then you will need to create a copy of its secret in the namespace you use for workflows.

## 7. Run a workflow which uses artifacts
```
$ argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/artifact-passing.yaml
```

## 8. Access the Argo UI

By default, the Argo UI service is not exposed with an external IP. To access the UI, use one of the following methods:

#### Method 1: kubectl port-forward
Run:
```
$ kubectl port-forward $(kubectl get pods -n kube-system -l app=argo-ui -o jsonpath='{.items[0].metadata.name}') -n kube-system 8001:8001
```
Then visit: http://127.0.0.1:8001/

#### Method 2: kubectl proxy
Run:
```
$ kubectl proxy
```
Then visit: http://127.0.0.1:8001/api/v1/proxy/namespaces/kube-system/services/argo-ui/

NOTE: artifact download and webconsole is not supported using this method

#### Method 3: Use a LoadBalancer

Update the argo-ui service to be of type `LoadBalancer`.
```
$ kubectl patch svc argo-ui -n kube-system -p '{"spec": {"type": "LoadBalancer"}}'
```
Then wait for the external IP to be made available:
```
$ kubectl get svc argo-ui -n kube-system
NAME      TYPE           CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
argo-ui   LoadBalancer   10.19.255.205   35.197.49.167   80:30999/TCP   1m
```

NOTE: On Minikube, you won't get an external IP after updating the service -- it will always show `pending`. Run the following command to determine the Argo UI URL:
```
$ minikube service -n kube-system --url argo-ui
```
