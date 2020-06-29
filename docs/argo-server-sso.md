# Argo Server SSO

![alpha](assets/alpha.svg)

> v2.9 and after

## To start Argo Server with SSO.

Firstly, configure the settings [workflow-controller-configmap.yaml](workflow-controller-configmap.yaml) with the correct OAuth 2 values.

Then, start the Argo Server using the SSO [auth mode](argo-server-auth-mode.md):

```
argo server --auth-mode sso --auth-mode ...
```
