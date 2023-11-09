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
* A **managed namespace install**: only executes workflows in a separate namespace from the one it is installed in. See [Managed Namespace](managed-namespace.md) for more details.

## Additional installation considerations

Review the following:

* [Security](security.md).
* [Scaling](scaling.md) and [running at massive scale](running-at-massive-scale.md).
* [High-availability](high-availability.md)
* [Disaster recovery](disaster-recovery.md)
