# Argo Install Manifests

Two 
| File | Description |
|------|-------------|
| install.yaml | Standard argo cluster-wide installation. Controller operates on all namespaces |
| namespace-install.yaml | Installation of argo which operates on a single namespace. Controller does not require to be run with clusterrole |