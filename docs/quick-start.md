# Quick Start

To see how Argo Workflows work, you can install it and run examples of simple workflows and workflows that use artifacts.

Firstly, you'll need a Kubernetes cluster and `kubectl` set-up

## Install Argo Workflows

To get started quickly, you can use the quick start manifest which will install Argo Workflow as well as some commonly used components:

!!! note
    These manifests are intended to help you get started quickly. They are not suitable in production, on test environments, or any environment containing any real data. They contain hard-coded passwords that are publicly available.

```sh
kubectl create ns argo
kubectl apply -n argo -f https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start-postgres.yaml
```

!!! note
    On GKE, you may need to grant your account the ability to create new `clusterrole`s

```sh
kubectl create clusterrolebinding YOURNAME-cluster-admin-binding --clusterrole=cluster-admin --user=YOUREMAIL@gmail.com
```

!!! note
    To run Argo on GKE Autopilot, you must use the `emissary` executor or the `k8sapi` executor. Find more information on our [executors doc](workflow-executors.md).

If you are running Argo Workflows locally (e.g. using Minikube or Docker for Desktop), open a port-forward so you can access the namespace:

```sh
kubectl -n argo port-forward deployment/argo-server 2746:2746
```

This will serve the user interface on https://localhost:2746

If you're using running Argo Workflows on a remote cluster (e.g. on EKS or GKE) then [follow these instructions](argo-server.md#access-the-argo-workflows-ui). 

Next, Download the latest Argo CLI from our [releases page](https://github.com/argoproj/argo-workflows/releases/latest).

Finally, submit an example workflow:  

`argo submit -n argo --watch https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/hello-world.yaml`

The `--watch` flag used above will allow you to observe the workflow as it runs and the status of whether it succeeds. 
When the workflow completes, the watch on the workflow will stop.

You can list all the Workflows you have submitted by running the command below:

`argo list -n argo`

You will notice the Workflow name has a `hello-world-` prefix followed by random characters. These characters are used 
to give Workflows unique names to help identify specific runs of a Workflow. If you submitted this Workflow again, 
the next Workflow run would have a different name.

Using the `argo get` command, you can always review details of a Workflow run. The output for the command below will 
be the same as the information shown as when you submitted the Workflow:

`argo get -n argo @latest`

The `@latest` argument to the CLI is a short cut to view the latest Workflow run that was executed. 

You can also observe the logs of the Workflow run by running the following:

`argo logs -n argo @latest`

Now that you have understanding of using Workflows, you can check out other [Workflow examples](https://github.com/argoproj/argo-workflows/blob/master/examples/README.md) to see additional uses of Worklows.
