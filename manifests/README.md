# Argo Install Manifests

Several sets of manifests are provided:

| File | Description |
|------|-------------|
| [install.yaml](install.yaml) | Standard argo cluster-wide installation. Controller operates on all namespaces. |
| [namespace-install.yaml](namespace-install.yaml) | Installation of argo which operates on a single namespace. Controller does not require to be run with clusterrole. Installs to `argo` namespace as an example. |
| [quick-start-minimal.yaml](quick-start-minimal.yaml) | Quick start including MinIO. Suitable for testing. |
| [quick-start-mysql.yaml](quick-start-mysql.yaml) | Quick start including MinIO and MySQL. Suitable for testing. |
| [quick-start-postgres.yaml](quick-start-postgres.yaml) | Quick start including MinIO and Postgres. Suitable for testing. |

If installing with `kubectl apply -f https://...`, remember to use the link to the file's raw version.
Otherwise you will get `mapping values are not allowed in this context`.

## Kustomize 

> v2.8 and after

You can use `./cluster-install` and `./namespace-install` as Kustomize remote bases.

## Helm

See [argo-helm](https://github.com/argoproj/argo-helm).