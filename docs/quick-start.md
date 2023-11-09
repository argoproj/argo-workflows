# Quick Start

To see how Argo Workflows work, you can install it and run examples of simple workflows.

Before you start you need a Kubernetes cluster and `kubectl` set up to be able to access that cluster. For the purposes of getting up and running, a local cluster is fine. You could consider the following local Kubernetes cluster options:

* [minikube](https://minikube.sigs.k8s.io/docs/)
* [kind](https://kind.sigs.k8s.io/)
* [k3s](https://k3s.io/) or [k3d](https://k3d.io/)
* [Docker Desktop](https://www.docker.com/products/docker-desktop/)

Alternatively, if you want to try out Argo Workflows and don't want to set up a Kubernetes cluster, try the [Killercoda course](training.md#hands-on).

!!! Warning "Development vs. Production"
    These instructions are intended to help you get started quickly. They are not suitable in production. For production installs, please refer to [the installation documentation](installation.md).

## Install Argo Workflows

To install Argo Workflows, navigate to the [releases page](https://github.com/argoproj/argo-workflows/releases/latest) and find the release you wish to use (the latest full release is preferred).

Scroll down to the `Controller and Server` section and execute the `kubectl` commands.

Below is an example of the install commands, ensure that you update the command to install the correct version number:

```yaml
kubectl create namespace argo
kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/download/v<<ARGO_WORKFLOWS_VERSION>>/install.yaml
```

### Patch argo-server authentication

The argo-server (and thus the UI) defaults to client authentication, which requires clients to provide their Kubernetes bearer token in order to authenticate. For more information, refer to the [Argo Server Auth Mode documentation](argo-server-auth-mode.md). We will switch the authentication mode to `server` so that we can bypass the UI login for now:

```bash
kubectl patch deployment \
  argo-server \
  --namespace argo \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": [
  "server",
  "--auth-mode=server"
]}]'

```

### Port-forward the UI

Open a port-forward so you can access the UI:

```bash
kubectl -n argo port-forward deployment/argo-server 2746:2746
```

This will serve the UI on <https://localhost:2746>. Due to the self-signed certificate, you will receive a TLS error which you will need to manually approve.

> Pay close attention to the URI. It uses `https` and not `http`. Navigating to `http://localhost:2746` result in server-side error that breaks the port-forwarding.

## Install the Argo Workflows CLI

You can more easily interact with Argo Workflows with the [Argo CLI](walk-through/argo-cli.md).

## Submitting an example workflow

### Submit an example workflow (CLI)

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

### Submit an example workflow (GUI)

* Open a port-forward so you can access the UI:

```bash
kubectl -n argo port-forward deployment/argo-server 2746:2746
```

* Navigate your browser to <https://localhost:2746>.

* Click `+ Submit New Workflow` and then `Edit using full workflow options`

* You can find an example workflow already in the text field. Press `+ Create` to start the workflow.
