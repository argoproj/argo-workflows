# CRD Installer Image

> v4.0.9 and 4.1.0 and after

The `argo-workflows-crdinstaller` image installs the [full CRDs](installation.md#full-crds) into a cluster.
It bundles `kubectl` together with the full CRD manifests for exactly the Argo Workflows version the image is tagged with, so it can install them without any network access to GitHub — including in air-gapped clusters.

It is published to `quay.io/argoproj/argo-workflows-crdinstaller` with the same tags as the other Argo Workflows images: one tag per release (for example `v4.1.0`), plus `latest` tracking the tip of `main`.

The image is not used by the official release manifests, which include the CRDs directly.
It exists for installation tooling that cannot apply the full CRDs itself — most notably Helm, which cannot server-side apply CRDs of this size from within a chart.
The [community Helm chart](https://github.com/argoproj/argo-helm) runs it as a pre-install/pre-upgrade hook Job.

## Contents

- `/bin/kubectl` — the image is based on the official [`registry.k8s.io/kubectl`](https://registry.k8s.io/) image, and `kubectl` is the entrypoint.
- `/crds/full/` — the [full CRDs](https://github.com/argoproj/argo-workflows/tree/main/manifests/base/crds/full) matching the image's version.

The default command is:

```bash
kubectl apply --server-side --force-conflicts -v=6 -f /crds/full/
```

[Server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) is required because the full CRDs exceed the size limit of the `last-applied-configuration` annotation used by client-side apply.
`-v=6` logs each API request `kubectl` makes, so the Job's logs record which CRDs it applied.

You can override the arguments, for example to preview changes with `apply --server-side --dry-run=server -f /crds/full/`.

The container runs as a non-root user (UID 8737), like the other Argo Workflows images.

## Usage

The image expects the credentials it runs with to be permitted to manage CRDs; it does not create any RBAC itself.
Run it as a Kubernetes Job with a ServiceAccount bound to a ClusterRole such as the one below:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: argo-workflows-crd-installer
  namespace: argo
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: argo-workflows-crd-installer
rules:
  - apiGroups: [apiextensions.k8s.io]
    resources: [customresourcedefinitions]
    verbs: [create, get, list, patch, update]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: argo-workflows-crd-installer
subjects:
  - kind: ServiceAccount
    name: argo-workflows-crd-installer
    namespace: argo
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: argo-workflows-crd-installer
---
apiVersion: batch/v1
kind: Job
metadata:
  name: argo-workflows-crd-installer
  namespace: argo
spec:
  template:
    spec:
      serviceAccountName: argo-workflows-crd-installer
      containers:
        - name: install
          image: quay.io/argoproj/argo-workflows-crdinstaller:v4.1.0
      restartPolicy: Never
  backoffLimit: 3
```

Use the image tag matching the Argo Workflows version you are installing, so the CRDs stay in sync with the controller and server.
