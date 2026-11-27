Description: New `argo-workflows-crdinstaller` image for installing the full CRDs
Authors: [Alan Clucas](https://github.com/Joibel)
Component: General
Issues: 16621

A new image, `quay.io/argoproj/argo-workflows-crdinstaller`, is published with each release.
It bundles `kubectl` and the [full CRDs](https://argo-workflows.readthedocs.io/en/latest/installation/#full-crds) matching the release, and by default applies them using server-side apply.
It is intended to run as a Kubernetes Job by installation tooling that cannot server-side apply the full CRDs itself, such as the community Helm chart, and works without network access to GitHub — including in air-gapped clusters.
See [the CRD installer documentation](https://argo-workflows.readthedocs.io/en/latest/crd-installer/) for usage and the required RBAC.
