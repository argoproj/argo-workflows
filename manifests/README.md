# Argo Install Manifests

Two sets of manifests are provided:

| File | Description |
|------|-------------|
| [install.yaml](install.yaml) | Standard argo cluster-wide installation. Controller operates on all namespaces |
| [namespace-install.yaml](namespace-install.yaml) | Installation of argo which operates on a single namespace. Controller does not require to be run with clusterrole. Installs to `argo` namespace as an example. |
| [quick-start-mysql.yaml](quick-start-mysql.yaml) | Quick start including MinIO and MySQL. Suitable for testing. |
| [quick-start-no-db.yaml](quick-start-no-db.yaml) | Quick start including MinIO. Suitable for testing. |
| [quick-start-postgres.yaml](quick-start-postgres.yaml) | Quick start including MinIO and Postgres. Suitable for testing. |
