# Quick Start

To try out Argo Workflows, you can install it and run example workflows.

Alternatively, if you don't want to set up a Kubernetes cluster, try the [Killercoda course](training.md#hands-on).

## Prerequisites

Before installing Argo, you need a Kubernetes cluster and `kubectl` configured to access it.
For quick testing, you can use a local cluster with:

* [minikube](https://minikube.sigs.k8s.io/docs/)
* [kind](https://kind.sigs.k8s.io/)
* [k3s](https://k3s.io/) or [k3d](https://k3d.io/)
* [Docker Desktop](https://www.docker.com/products/docker-desktop/)

!!! Warning "Development vs. Production"
    These instructions are intended to help you get started quickly. They are not suitable for production.
    For production installs, please refer to [the installation documentation](installation.md).

## Install Argo Workflows

To install Argo Workflows, go to the [releases page](https://github.com/argoproj/argo-workflows/releases/latest).
Scroll down to the `Controller and Server` section and execute the `kubectl` commands.

Below is an example of the install commands (substitute `<<ARGO_WORKFLOWS_VERSION>>` with the version number):

```bash
kubectl create namespace argo
kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/download/v<<ARGO_WORKFLOWS_VERSION>>/quick-start-minimal.yaml
```

## Install the Argo Workflows CLI

You can more easily interact with Argo Workflows with the [Argo CLI](walk-through/argo-cli.md).

## Submit an example workflow

### Submit via the CLI

```bash
argo submit -n argo --watch https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/hello-world.yaml
```

The `--watch` flag watches the workflow as it runs and reports whether it succeeds or not.
When the workflow completes, the watch stops.

You can list all submitted Workflows by running the command below:

```bash
argo list -n argo
```

The Workflow name has a `hello-world-` prefix followed by random characters.
These characters give Workflows unique names to help identify specific runs of a Workflow.
If you submit this Workflow again, the next run will have different characters.

You can review the details of a Workflow run using the `argo get` command.
The output for the command below will be the same as the information shown when you submitted the Workflow:

```bash
argo get -n argo @latest
```

The `@latest` argument is a shortcut to view the latest Workflow run.

You can observe the logs of the Workflow run with the following command:

```bash
argo logs -n argo @latest
```

### Submit via the UI

1. Forward the Server's port to access the UI:

    ```bash
    kubectl -n argo port-forward service/argo-server 2746:2746
    ```

1. Navigate your browser to <https://localhost:2746>.
    * **Note**: The URL uses `https` and not `http`. Navigating to `http` will result in a server-side error.
    * Due to the self-signed certificate, you will receive a TLS error which you will need to manually approve.
1. Click `+ Submit New Workflow` and then `Edit using full workflow options`
1. You can find an example workflow already in the text field. Press `+ Create` to start the workflow.
