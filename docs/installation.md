# Installation

You can choose one of three common installations:

* **cluster install** Execute workflows in any namespace?
* **namespace install** Only execute workflows in the same namespace we install in (typically `argo`).
* **managed namespace install**: Only execute workflows in a specific namespace ([learn more](managed-namespace.md)).

## Recommendations

* Make sure you're using the manifests attached to each Github release. See [this link](https://github.com/argoproj/argo-workflows/releases/latest) for the most recent manifests.
* Read the [upgrading guide](upgrading.md) before any major upgrade to be aware of breaking changes.
* If you are using GitOps, copy the manifests into your repository, rather use `base` to reference them. Fewer issues
  regarding network loss, and you'll be able to check what has changed each time you upgrade.
* Double-check you have the right version of your executor configured, it's easy to miss.




