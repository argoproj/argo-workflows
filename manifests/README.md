# Argo Install Manifests

Two sets of manifests are provided:

| File | Description |
|------|-------------|
| [install.yaml](install.yaml) | Standard argo cluster-wide installation. Controller operates on all namespaces |
| [namespace-install.yaml](namespace-install.yaml) | Installation of argo which operates on a single namespace. Controller does not require to be run with clusterrole. Installs to `argo` namespace as an example. |
