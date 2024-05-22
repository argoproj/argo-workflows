# FAQ

## "token not valid", "any bearer token is able to login in the UI or use the API"

You may not have configured Argo Server authentication correctly.

If you want SSO, try running with `--auth-mode=sso`.
If you're using `--auth-mode=client`, make sure you have `Bearer` in front of the ServiceAccount Secret, as mentioned in [Access Token](access-token.md#token-creation).

[Learn more about the Argo Server set-up](argo-server.md)

## Argo Server return EOF error

Since v3.0 the Argo Server listens for HTTPS requests, rather than HTTP. Try changing your URL to HTTPS, or start Argo Server using `--secure=false`.

## My workflow hangs

Check your `wait` container logs:

Is there an RBAC error?

[Learn more about workflow RBAC](workflow-rbac.md)

## `cannot patch resource "pods" in API group ""` error

You're probably getting a permission denied error because your RBAC is not configured.

[Learn more about workflow RBAC](workflow-rbac.md)
