# Argo Server Auth Mode

You can choose which kube config the Argo Server uses:

* "server" - in hosted mode, use the kube config of service account, in local mode, use your local kube config.
* "client" - requires clients to provide their Kubernetes bearer token and use that.
* "hybrid" - use the client token if provided, fallback to the server token if note.
* "sso" - since v2.9, use single sign-on, this will use the same service account as per "server" for RBAC. We expect to change this in the future so that the OAuth claims are mapped to service accounts.

By default, the server will start with auth mode of "server".
