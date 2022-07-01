# Quick Start

To see how Argo Workflows work, you can install it and run examples of simple workflows and workflows that use artifacts.

Before you start you need a Kubernetes cluster and `kubectl` set-up

## Install Argo Workflows

To get started quickly, you can use the quick start manifest which will install Argo Workflows as well as some commonly used components:

⚠️ These manifests are intended to help you get started quickly. They are not suitable in production. They contain hard-coded passwords that are publicly available.

```bash
kubectl create ns argo
kubectl apply -n argo -f https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start-postgres.yaml
```

Open a port-forward so you can access the UI:

```bash
kubectl -n argo port-forward deployment/argo-server 2746:2746
```

This will serve the UI on <https://localhost:2746>

Next, Download the latest Argo CLI from our [releases page](https://github.com/argoproj/argo-workflows/releases/latest).

Finally, submit an example workflow:  

```bash
argo submit -n argo --watch https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/hello-world.yaml
```

The `--watch` flag used above will allow you to observe the workflow as it runs and the status of whether it succeeds.
When the workflow completes, the watch on the workflow will stop.

You can list all the Workflows you have submitted by running the command below:

```bash
argo list -n argo
```

You will notice the Workflow name has a `hello-world-` prefix followed by random characters. These characters are used
to give Workflows unique names to help identify specific runs of a Workflow. If you submitted this Workflow again,
the next Workflow run would have a different name.

Using the `argo get` command, you can always review details of a Workflow run. The output for the command below will
be the same as the information shown as when you submitted the Workflow:

```bash
argo get -n argo @latest
```

The `@latest` argument to the CLI is a short cut to view the latest Workflow run that was executed.

You can also observe the logs of the Workflow run by running the following:

```bash
argo logs -n argo @latest
```

💡 If you want to try out Argo Workflows and don't want to set up a Kubernetes cluster, the community is working on a replacement for the old Katacoda course since
Katacoda was shut down. Please give a thumbs up or comment on [this issue](https://github.com/argoproj/argo-workflows/issues/8899) with your support and feedback.
