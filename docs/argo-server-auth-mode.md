# Argo Server Auth Mode

You can configure how the Argo Server authenticates to Kubernetes:

* `server`: In [hosted mode](argo-server.md#hosted-mode), use the Server's Service Account. In [local mode](argo-server.md#local-mode), use your local kube config.
* `client`: Use the Kubernetes [bearer token of clients](access-token.md). The Server validates the token with a [`SelfSubjectReview`](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#self-subject-review) before serving a request. Successful reviews are cached for `ARGO_SERVER_TOKEN_REVIEW_CACHE_TTL` (default `1m`) to limit API server load.
* `sso`: Use [single sign-on](argo-server-sso.md). This will use the same SA as `server` for RBAC, unless you have enabled [SSO RBAC](argo-server-sso.md#sso-rbac)

For v3.0 and after, the default is `client`. Prior to v3.0, it was `server`.

To configure the Server's auth modes, use one or multiple `--auth-mode` flags. For example:

```bash
argo server --auth-mode=sso --auth-mode=client
```

On Kubernetes versions without `SelfSubjectReview`, bearer authentication can use
the stable `TokenReview` API. Grant the Argo Server's ServiceAccount only `create`
on `tokenreviews.authentication.k8s.io` with a ClusterRole and ClusterRoleBinding.
This fallback runs only when `SelfSubjectReview` returns a not-found response, or when a
Forbidden response is accompanied by API discovery proving the API is absent.
Denials for an available API, discovery failures and other errors are not
bypassed. The token must be authenticated and
identify a non-anonymous user. Subsequent resource requests still use the
caller's credentials and permissions, not the Server's ServiceAccount.
Basic authentication still requires `SelfSubjectReview` support.
