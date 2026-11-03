# Service Accounts

## Configure the service account to run Workflows

### Roles, Role-Bindings, and Service Accounts

To support features such as artifacts, outputs, and access to secrets, Argo needs to communicate with Kubernetes resources using the Kubernetes API.
To communicate with the Kubernetes API, Argo uses a `ServiceAccount` to authenticate itself to the Kubernetes API.
You can specify which `Role` (that is, which permissions) the `ServiceAccount` that Argo uses by binding a `Role` to a `ServiceAccount` using a `RoleBinding`.

Then, when submitting Workflows, you can specify which `ServiceAccount` Argo uses:

```bash
argo submit --serviceaccount <name>
```

If you don't specify a `ServiceAccount`, Argo uses the `default` `ServiceAccount` from the namespace the workflow runs in, which almost always has insufficient privileges by default.

For more information about granting Argo the necessary permissions for your use case, see [Workflow RBAC](workflow-rbac.md).

### Granting admin privileges

For the purposes of this demo, this guide grants the `default` `ServiceAccount` admin privileges (that is, it binds the `admin` `Role` to the `default` `ServiceAccount` of the current namespace):

```bash
kubectl create rolebinding default-admin --clusterrole=admin --serviceaccount=argo:default -n argo
```

**Note that this grants admin privileges to the `default` `ServiceAccount` in the namespace where you run the command, so you can only run Workflows in the namespace where you created the `RoleBinding`.**
