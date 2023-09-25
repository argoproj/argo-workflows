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

Breaking changes will be documented in [upgrading guide](upgrading.md).

## Supported Version Skew

Both the `argo-server` and `argocli` should be the same version as the controller.

## Release Cycle

New minor versions are released roughly every 6 months. Release candidates for each major release are typically available
for 4-6 weeks before the release becomes generally available.

Otherwise, we typically release every two weeks:

* Patch fixes for the current stable version.
* The next release candidate, if we are currently in a release-cycle.

## Kubernetes Compatibility Matrix

| Argo Workflows \ Kubernetes | 1.17 | 1.18 | 1.19 | 1.20 | 1.21 | 1.22 | 1.23 | 1.24 | 1.25 | 1.26 | 1.27 |
|-----------------------|------|------|------|------|------|------|------|------|------|------|------|
| **3.4**           | `x` | `x` | `x` | `?` | `✓` | `✓` | `✓` | `✓` | `✓` | `✓` | `✓` |
| **3.3**           | `?` | `?` | `?` | `?` | `✓` | `✓` | `✓` | `?` | `?` | `?` | `?` |
| **3.2**           | `?` | `?` | `✓` | `✓` | `✓` | `?` | `?` | `?` | `?` | `?` | `?` |
| **3.1**           | `✓` | `✓` | `✓` | `?` | `?` | `?` | `?` | `?` | `?` | `?` | `?` |

* `✓` Fully supported versions.
* `?` Due to breaking changes might not work. Also, we haven't thoroughly tested against this version.
* `✕` Unsupported versions.

### Notes on Compatibility

Argo versions may be compatible with newer and older versions than what it is listed but only three minor versions are supported per Argo release unless otherwise noted.

The main branch of `Argo Workflows` is currently tested on `Kubernetes` 1.27.
