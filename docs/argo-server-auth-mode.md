# Argo Server Auth Mode

You can choose which kube config authorization the Argo Server uses to access the Kubernetes API:

* `server` - in hosted mode, use the kube config of service account, in local mode, use your local kube config.
* `client` - requires clients to provide their Kubernetes bearer token and use that.
* [`sso`](./argo-server-sso.md) - (since v2.9) use OIDC to authenticate users with single-sign-on.
   * [SSO Impersonate](./argo-server-sso.md#sso-impersonate) - use Kubernetes [SubjectAccessReviews](https://kubernetes.io/docs/reference/access-authn-authz/authorization/#checking-api-access) with a User extracted from a OIDC JWT claim 
   * [SSO RBAC](./argo-server-sso.md#sso-rbac) - use annotations on Kubernetes ServiceAccounts to select OIDC JWT groups

The server used to start with auth mode of "server" by default, but since v3.0 it defaults to the "client".

To change the server auth mode specify the list as multiple auth-mode flags:
```shell
argo server --auth-mode sso --auth-mode ...
```
