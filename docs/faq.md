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

You're probably getting a permission denied error because your RBAC is not configured, learn more about workflow RBAC.

There are workflow service accounts used to execute the workflow’s pods:

1. User service accounts specified by users.
2. Accounts used by apps (e.g. Jenkins).
3. The default service account, used when none is specified.

We do not recommend you use the default service account. You might need your pods to have special permissions, so you’d have to escalate its privileges.

If Argo Workflows is set-up correctly, e.g. using the manifests provided with it, then you only need to concern yourself with the user service accounts and the default service account.

To set-up a service account for your workflow, you need to create three things:

1. A role with the correct permissions.
2. A service account.
3. A role binding between the service account and the role.

[For more details](https://blog.argoproj.io/demystifying-argo-workflowss-kubernetes-rbac-7a1406d446fc)

> There is an error about /var/run/docker.sock.

Try using a different container runtime executor.

[Learn more about executors](workflow-executors.md)