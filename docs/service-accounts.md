# Service Accounts

## Configure the service account to run Workflows

### Roles, Role-Bindings, and Service Accounts

In order for Argo to support features such as artifacts, outputs, access to secrets, etc. it needs to communicate with Kubernetes resources
using the Kubernetes API. To communicate with the Kubernetes API, Argo uses a `ServiceAccount` to authenticate itself to the Kubernetes API.
You can specify which `Role` (i.e. which permissions) the `ServiceAccount` that Argo uses by binding a `Role` to a `ServiceAccount` using a `RoleBinding`

Then, when submitting Workflows you can specify which `ServiceAccount` Argo uses using:

```bash
argo submit --serviceaccount <name>
```

When no `ServiceAccount` is provided, Argo will use the `default` `ServiceAccount` from the namespace from which it is run, which will almost always have insufficient privileges by default.

For more information about granting Argo the necessary permissions for your use case see [Workflow RBAC](workflow-rbac.md).

### Granting admin privileges

For the purposes of this demo, we will grant the `default` `ServiceAccount` admin privileges (i.e., we will bind the `admin` `Role` to the `default` `ServiceAccount` of the current namespace):

```bash
kubectl create rolebinding default-admin --clusterrole=admin --serviceaccount=argo:default -n argo
```

**Note that this will grant admin privileges to the `default` `ServiceAccount` in the namespace that the command is run from, so you will only be able to
run Workflows in the namespace where the `RoleBinding` was made.**
