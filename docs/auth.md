# Auth

You can choose which kube config the Argo Server uses:

* "server" - in hosted mode, use the kube config of service account, in local mode, use your local kube config.
* "client" - requires clients to provide their Kubernetes bearer token and use that.
* "hybrid" - use the client token if provided, fallback to the server token if note.
* "sso" - use single sign-on, this will use your Argo Server's service account as per "server".

By default, the server will start with auth mode of "server".

## Single sign-on (SSO)

![alpha](assets/alpha.svg)

> v2.9 and after

SSO allows you to use your single sign-on provider to login.

This does not provide an RBAC today. Your user will have by fully escalted to the permissions of Argo Server's service account.
 
The CLI does not support SSO. 