# Argo Server SSO

![alpha](assets/alpha.svg)

> v2.9 and after

## To start Argo Server with SSO.

First, creat a [secret](../manifests/quick-start/base/argo-server-oauth2-secret.yaml) with the correct OAuth 2 values.

Secondly, start using the SSO [auth mode](argo-server-auth-mode.md):

```
argo server --auth-mode sso --auth-mode ...
```

Finally, you probably want to enable [RBAC](argo-server-rbac.md).