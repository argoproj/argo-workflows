# Releases

## Supported Versions

Versions are expressed as x.y.z, where x is the major version, y is the minor version, and z is the patch version,
following Semantic Versioning terminology.

We maintain release branches for the most recent two minor releases.

Fixes may be backported to release branches, depending on severity, risk, and, feasibility.

If a release contains breaking changes, or CVE fixes, this will documented in the release notes.

## Supported Version Skew

Both the `argo-server` and `argocli` should be the same version as the controller.

# Release Cycle

For **stable**, use the latest patch version.
For **unstable**, we build and tag `latest` images for every commit to master.

New minor versions are released roughly every 3 months. Release candidates for each major release are typically available
for 4-6 weeks before the release becomes generally available.

Otherwise, we typically release weekly:

* Patch fixes for the current stable version. 
* The next release candidate, if we are currently in a release-cycle.
