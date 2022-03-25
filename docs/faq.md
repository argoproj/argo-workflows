# FAQ

> "token not valid for running mode", "any bearer token is able to login in the UI or use the API"

You've not configured Argo Server authentication correctly. If you want SSO, try running with `--auth-mode=sso`.

[Learn more about the Argo Server set-up](argo-server.md)

> Argo Server return EOF error

Since v3.0 the Argo Server listens for HTTPS requests, rather than HTTP. Try changing your URL to HTTPS, or start Argo Server using `--secure=false`.

> My workflow hangs

Check your `wait` container logs:

Is there an RBAC error?

[Learn more about workflow RBAC](workflow-rbac.md)

> Return "unknown (get pods)" error

You're probably getting a permission denied error because your RBAC is not configured.

[Learn more about workflow RBAC](workflow-rbac.md) and [even more details](https://blog.argoproj.io/demystifying-argo-workflowss-kubernetes-rbac-7a1406d446fc)

> There is an error about /var/run/docker.sock.

Try using a different container runtime executor.

[Learn more about executors](workflow-executors.md)
