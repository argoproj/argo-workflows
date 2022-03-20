# Installation

## Argo on Desktop

Use the [quick-start manifests](quick-start.md).

## Argo in Production

Determine your base installation option.

* A **cluster install** will watch and execute workflows in all namespaces.
* A **namespace install** only executes workflows in the namespace it is installed in (typically `argo`).
* A **managed namespace install**: only executes workflows in a specific namespace ([learn more](managed-namespace.md)).

⚠️ `latest` is tip, not stable. Never run it. Make sure you're using the manifests attached to each Github release. See [this link](https://github.com/argoproj/argo-workflows/releases/latest) for the most recent manifests. 

⚠️ Double-check you have the right version of your executor configured, it's easy to miss.

⚠️ If you are using GitOps. Never use Kustomize remote base, this is dangerous. Instead, copy the manifests into your Git repo.

Review the following:

 * [Security](security.md).
 * [Scaling](scaling.md) and [running at massive scale](running-at-massive-scale.md).
 * [High-availability](high-availability.md)
 * [Disaster recovery](disaster-recovery.md)

Read the [upgrading guide](upgrading.md) before any major upgrade to be aware of breaking changes.


