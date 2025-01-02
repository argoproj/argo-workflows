# Installation

## Non-production installation

If you just want to try out Argo Workflows in a non-production environment (including on desktop via minikube/kind/k3d etc) follow the [quick-start guide](quick-start.md).

## Production installation

### Installation Methods

#### Official release manifests

To install Argo Workflows, navigate to the [releases page](https://github.com/argoproj/argo-workflows/releases/latest) and find the release you wish to use (the latest full release is preferred). Scroll down to the `Controller and Server` section and execute the `kubectl` commands.

You can use Kustomize to patch your preferred [configurations](managed-namespace.md) on top of the base manifest.

!!! Note "Use a full hash"
    If you are using a [remote base](https://github.com/kubernetes-sigs/kustomize/blob/ab519fdc13ded9875e42d70ac8a5b1b9023a2dbb/examples/remoteBuild.md) with Kustomize, you should specify a full commit hash, for example `?ref=960af331a8c0a3f2e263c8b90f1daf4303816ba8`.

!!! Warning "`latest` vs stable"
    `latest` is the tip of the `main` branch and may not be stable.
    In production, you should use a specific release version.

#### Argo Workflows Helm Chart

You can install Argo Workflows using the community maintained [Helm charts](https://github.com/argoproj/argo-helm).

### Full CRDs

The official release manifests come with stripped-down CRDs that omit validation information.
This is a workaround for [Kubernetes size limitations](https://github.com/kubernetes/kubernetes/issues/82292) when using client-side apply.
As of version 3.7, the full CRDs can be installed using [server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) via the following command:

```bash
kubectl apply --server-side --kustomize https://github.com/argoproj/argo-workflows/manifests/base/crds/full?ref=v3.7.0
```

## Installation options

Determine your base installation option.

* A **cluster install** will watch and execute workflows in all namespaces. This is the default installation option when installing using the official release manifests.
* A **namespace install** only executes workflows in the namespace it is installed in (typically `argo`). Look for `namespace-install.yaml` in the [release assets](https://github.com/argoproj/argo-workflows/releases/latest).
* A **managed namespace install**: only executes workflows in a separate namespace from the one it is installed in. See [Managed Namespace](managed-namespace.md) for more details.

## Additional installation considerations

Review the following:

* [Workflow RBAC](workflow-rbac.md)
* [Security](security.md).
* [Scaling](scaling.md) and [running at massive scale](running-at-massive-scale.md).
* [High-availability](high-availability.md)
* [Disaster recovery](disaster-recovery.md)
