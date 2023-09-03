# Installation

## Non-production installation

If you just want to try out Argo Workflows in a non-production environment (including on desktop via minikube/kind/k3d etc) follow the [quick-start guide](quick-start.md).

## Production installation

### Installation Methods

#### Official release manifests

To install Argo Workflows, navigate to the [releases page](https://github.com/argoproj/argo-workflows/releases/latest) and find the release you wish to use (the latest full release is preferred). Scroll down to the `Controller and Server` section and execute the `kubectl` commands.

You can use Kustomize to patch your preferred [configurations](managed-namespace.md) on top of the base manifest.

⚠️ If you are using GitOps, never use Kustomize remote base: this is dangerous. Instead, copy the manifests into your Git repo.

⚠️ `latest` is tip, not stable. Never run it in production.

#### Argo Workflows Helm Chart

You can install Argo Workflows using the community maintained [Helm charts](https://github.com/argoproj/argo-helm).

## Installation options

Determine your base installation option.

* A **cluster install** will watch and execute workflows in all namespaces. This is the default installation option when installing using the official release manifests.
* A **namespace install** only executes workflows in the namespace it is installed in (typically `argo`). Look for `namespace-install.yaml` in the [release assets](https://github.com/argoproj/argo-workflows/releases/latest).
* A **managed namespace install**: only executes workflows in a specific namespace ([learn more](managed-namespace.md)).

!!! Warning "namespace install vs. managed namespace install"
    A namespace install allows Workflows to run only in the namespace where Argo Workflows is installed. A managed namespace install allows Workflows to run only in one namespace besides the one where Argo Workflows is installed. Using a managed namespace install might make sense if you want some users/processes to be able to run Workflows without granting them any privileges in the namespace where Argo Workflows is installed.  

    For example, if you only run CI/CD-related Workflows that are maintained by the same team that manages the Argo Workflows installation, it's probably reasonable to use a namespace install. But if all the Workflows are run by a separate data science team, it probably makes sense to give them a data-science-workflows namespace and run a "managed namespace install" of Argo Workflows from another namespace.
    To configure a managed namespace install, edit the workflow-controller and argo-server Deployments to pass the --managed-namespace argument.

## Additional installation considerations

Review the following:

* [Security](security.md).
* [Scaling](scaling.md) and [running at massive scale](running-at-massive-scale.md).
* [High-availability](high-availability.md)
* [Disaster recovery](disaster-recovery.md)
