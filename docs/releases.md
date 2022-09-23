# Releases

You can find the most recent version under [Github release](https://github.com/argoproj/argo-workflows/releases).

## Versioning

Versions are expressed as `x.y.z`, where `x` is the major version, `y` is the minor version, and `z` is the patch version,
following Semantic Versioning terminology.

Argo Workflows does not use Semantic Versioning. Minor versions may contain breaking changes. Patch versions only
contain bug fixes and minor features.

For **stable**, use the latest patch version.

⚠️ Read the [upgrading guide](upgrading.md) to find out about breaking changes before any upgrade.

## Supported Versions

We maintain release branches for the most recent two minor releases.

Fixes may be back-ported to release branches, depending on severity, risk, and, feasibility.

If a release contains breaking changes, or CVE fixes, this will documented in [upgrading guide](upgrading.md).

## Supported Version Skew

Both the `argo-server` and `argocli` should be the same version as the controller.

## Release Cycle

New minor versions are released roughly every 3 months. Release candidates for each major release are typically available
for 4-6 weeks before the release becomes generally available.

Otherwise, we typically release every two weeks:

* Patch fixes for the current stable version.
* The next release candidate, if we are currently in a release-cycle.
