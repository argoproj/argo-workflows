# Releases

## Supported Versions

Versions are expressed as x.y.z, where x is the major version, y is the minor version, and z is the patch version,
following Semantic Versioning terminology.

We maintain release branches for the most recent two minor releases.

Fixes may be backported to release branches, depending on severity, risk, and, feasibility.

If a release contains breaking changes, or CVE fixes, this will documented in the release notes.

## Supported Version Skew

Both the `argo-server` and `argocli` should be the same version as the controller.

## Supported Kubernetes Version

The compatible Argo workflow and Kubernetes versions are as below.

See [Kubernete version skew policy](https://kubernetes.io/releases/version-skew-policy/) for more deail.

|Argo workflow version|Kubernetes version|
|-|-|-|
|3.1|1.18-1.20|
|3.0|1.18-1.20|
|2.12|1.17-1.19|

# Release Cycle

For **unstable**, we build and tag `latest` images for every commit to master.

New major versions are released roughly every 3 months. Release candidates for each major release are typically available
for 6 weeks before the release becomes generally available.

Otherwise, we typically release once a week:

* Patch fixes for the current stable version. These are tagged `stable`.
* The next release candidate, if we are currently in a release-cycle.
