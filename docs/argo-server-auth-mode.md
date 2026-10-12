<!-- cspell:words tokenreviews -->
# Argo Server Auth Mode

You can configure how the Argo Server authenticates to Kubernetes:

* `server`: In [hosted mode](argo-server.md#hosted-mode), use the Server's Service Account. In [local mode](argo-server.md#local-mode), use your local kube config.
* `client`: Use the Kubernetes [bearer token of clients](access-token.md). The Server validates the token with a [`SelfSubjectReview`](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#self-subject-review) before serving a request. Successful reviews are cached for `ARGO_SERVER_TOKEN_REVIEW_CACHE_TTL` (default `1m`) to limit API server load.
* `sso`: Use [single sign-on](argo-server-sso.md). This will use the same SA as `server` for RBAC, unless you have enabled [SSO RBAC](argo-server-sso.md#sso-rbac)

The default is `client`.

To configure the Server's auth modes, use one or multiple `--auth-mode` flags. For example:

```bash
argo server --auth-mode=sso --auth-mode=client
```

On Kubernetes versions without `SelfSubjectReview`, the Server can validate bearer credentials through the stable `TokenReview` API.
Grant the Argo Server's ServiceAccount only `create` on `tokenreviews.authentication.k8s.io` with a ClusterRole and ClusterRoleBinding.
The Server uses this fallback when `SelfSubjectReview` returns not found, or when API discovery confirms that a Forbidden response refers to an absent API.
It rejects authorization denials for an available API, discovery failures and other errors.
The review must authenticate the token as a non-anonymous user.
Subsequent resource requests use the caller's credentials and permissions.
Basic authentication still requires `SelfSubjectReview` support.
