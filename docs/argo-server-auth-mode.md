# Argo Server Auth Mode

You can choose which kube config the Argo Server uses:

* `server` - in hosted mode, use the kube config of service account, in local mode, use your local kube config.
* `client` - requires clients to provide their Kubernetes bearer token and use that.
* [`sso`](./argo-server-sso.md) - since v2.9, use single sign-on, this will use the same service account as per "server" for RBAC. We expect to change this in the future so that the OAuth claims are mapped to service accounts.

The server used to start with auth mode of "server" by default, but since v3.0 it defaults to the "client".

To change the server auth mode specify the list as multiple auth-mode flags:

```bash
argo server --auth-mode=sso --auth-mode=...
```
