# Installation

You can choose one of three common installations:

* **cluster install** Execute workflows in any namespace? 
* **namespace install** Only execute workflows in the same namespace we install in (typically `argo`).
* **managed namespace install**: Only execute workflows in a specific namespace ([learn more](managed-namespace.md)).

Choose [a manifests from the list](https://github.com/argoproj/argo/tree/stable/manifests).

E.g.

```sh
kubectl create ns argo
kubectl apply -n argo -f https://raw.githubusercontent.com/argoproj/argo/stable/manifests/namespace-install.yaml 
```






