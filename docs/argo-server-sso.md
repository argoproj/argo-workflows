# Argo Server SSO

![GA](assets/ga.svg)

> v2.9 and after

## To start Argo Server with SSO.

Firstly, configure the settings [workflow-controller-configmap.yaml](workflow-controller-configmap.yaml) with the correct OAuth 2 values.

Next, create the k8s secrets for holding the OAuth2 `client-id` and `client-secret`. You may refer to the kubernetes documentation on [Managing secrets](https://kubernetes.io/docs/tasks/configmap-secret/). For example by using kubectl with literals:
```
kubectl create secret -n argo generic client-id-secret \
  --from-literal=client-id-key=foo

kubectl create secret -n argo generic client-secret-secret \
  --from-literal=client-secret-key=bar
```

Then, start the Argo Server using the SSO [auth mode](argo-server-auth-mode.md):

```
argo server --auth-mode sso --auth-mode ...
```
