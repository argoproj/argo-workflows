# Releases

You can find the most recent version in the [GitHub Releases](https://github.com/argoproj/argo-workflows/releases).

## Versioning

Versions are expressed as `x.y.z`, where `x` is the major version, `y` is the minor version, and `z` is the patch version, following Semantic Versioning terminology.

Argo Workflows does _not_ use Semantic Versioning.
Minor versions may contain breaking changes.
Patch versions only contain bug fixes and minor features.

For **stable**, use the latest patch version.

## Supported Versions

We maintain release branches for the most recent two minor releases.

Fixes may be backported to release branches, depending on severity, risk, and feasibility.

Breaking changes will be documented in the [upgrading guide](upgrading.md).

## Supported Version Skew

Both the `argo-server` and `argocli` should be the same version as the controller.

## Release Cycle

New minor versions are released roughly every 6 months.
Release Candidates (RCs) for major and minor versions are available roughly 4-6 weeks before General Availability (GA).

Features may be added in subsequent RCs.
If they are, the RC will be available for at least 2 weeks to ensure sufficient testing before GA.
If bugs are found with a feature and not resolved within the 2 week period, it will be rolled back and a new RC will be released before GA.

Otherwise, we typically release every two weeks:

* Patch fixes for the current stable version.
* The next RC, if we are currently in a release cycle.

## Tested Versions

--8<-- "docs/tested-kubernetes-versions.md"

### Notes on Compatibility

Argo versions may be compatible with newer and older Kubernetes versions, but only two minor versions are tested.

Note that Kubernetes [is backward compatible with clients](https://github.com/kubernetes/client-go/tree/aa7909e7d7c0661792ba21b9e882f3cd6ad0ce53?tab=readme-ov-file#compatibility-client-go---kubernetes-clusters), so newer k8s versions are generally supported.
The caveats with newer k8s versions are possible changes to experimental APIs and unused new features.
Argo uses stable Kubernetes APIs such as Pods and ConfigMaps; see the Controller and Server RBAC of your [installation](installation.md) for a full list.
