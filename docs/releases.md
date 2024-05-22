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

New minor versions are released roughly every 6 months.

Release candidates (RCs) for major and minor releases are typically available for 4-6 weeks before the release becomes generally available (GA). Features may be shipped in subsequent release candidates.

When features are shipped in a new release candidate, the most recent release candidate will be available for at least 2 weeks to ensure it is tested sufficiently before it is pushed to GA. If bugs are found with a feature and are not resolved within the 2 week period, the features will be rolled back so as to be saved for the next major/minor release timeline, and a new release candidate will be cut for testing before pushing to GA.

Otherwise, we typically release every two weeks:

* Patch fixes for the current stable version.
* The next release candidate, if we are currently in a release-cycle.

## Kubernetes Compatibility Matrix

| Argo Workflows \ Kubernetes | 1.26 | 1.27 | 1.28 | 1.29 | 1.30 |
|-----------------------|------|------|------|------|------|
| **3.5**               | `✓` | `✓` | `✓` | `✓` | `?` |
| **3.4**               | `✓` | `✓` | `?` | `?` | `?` |
| **3.3**               | `?` | `?` | `?` | `?` | `?` |

* `✓` Fully supported versions.
* `?` Due to breaking changes might not work. Also, we haven't thoroughly tested against this version.
* `✕` Unsupported versions.

### Notes on Compatibility

Argo versions may be compatible with newer and older versions than what it is listed but only three minor versions are supported per Argo release unless otherwise noted.

The main branch of `Argo Workflows` is currently tested on `Kubernetes` 1.29.
