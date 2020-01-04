# Getting Started

To see how Argo works, you can run examples of simple workflows and workflows that use artifacts.
For the latter, you'll set up an artifact repository for storing the artifacts that are passed in
the workflows. Here are the requirements and steps to run the workflows.

## Requirements

* Kubernetes 1.9 or later
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* Have a [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) file (default location is `~/.kube/config`)

Run the following to download and install Argo into the "argo" namespace:

```
curl -sfL http://bit.ly/get-argo | sh -
```

By default, this install in your cluster:

* MinIO - a free alternative to S3 - so you can experiment with saving artifacts.

NOTE: On GKE, you may need to grant your account the ability to create new `clusterrole`s

```
kubectl create clusterrolebinding YOURNAME-cluster-admin-binding --clusterrole=cluster-admin --user=YOUREMAIL@gmail.com
```

To access the UI, set-up a port-forward

```
kubectl -n argo port-forward svc/argo-server 2746:2746
```

You can now access the UI on http://localhost:2746.

## Run Sample Workflows From The CLI

```
export ARGO_SERVER=http://localhost:2746
# Run the Hello World workflows
argo submit --watch https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
argo submit --watch https://raw.githubusercontent.com/argoproj/argo/master/examples/coinflip.yaml
argo submit --watch https://raw.githubusercontent.com/argoproj/argo/master/examples/loops-maps.yaml
# Run a workflow that save artifacts:
argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/artifact-passing.yaml
argo list
argo get xxx-workflow-name-xxx
argo logs xxx-pod-name-xxx #from get command above
```

Additional examples and more information about the CLI are available on the [Argo Workflows by Example](../examples/README.md) page.

## Clean Up

Delete the "argo" namespace:

```
kubectl delete ns argo
```