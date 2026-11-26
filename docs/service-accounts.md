# Service Accounts

## Configure the service account to run Workflows

### Roles, Role-Bindings, and Service Accounts

Workflow pods use a `ServiceAccount` to authenticate to the Kubernetes API.
The executor in each pod uses it to report outputs, so at minimum the `ServiceAccount` needs `create` and `patch` on `workflowtaskresults`.
The executor reads artifact repository credentials from secrets mounted into the pod, so it does not need API access to those secrets.
Your own containers may need further permissions, for example to create Kubernetes resources.
You give a `ServiceAccount` its permissions by binding a `Role` to it with a `RoleBinding`.

You set the `ServiceAccount` that Workflow pods use with the `spec.serviceAccountName` field, or with the `--serviceaccount` flag when submitting:

```bash
argo submit --serviceaccount <name>
```

If none of the template, the Workflow or the [workflow defaults](default-workflow-specs.md) set `serviceAccountName`, Workflow pods use the `default` `ServiceAccount` from the namespace the Workflow runs in, which almost always has insufficient privileges by default.

For more information about granting Argo the necessary permissions for your use case, see [Workflow RBAC](workflow-rbac.md).

### Granting admin privileges

For this demo, grant admin privileges to the `default` `ServiceAccount` in the `argo` namespace by binding the `admin` `ClusterRole` to it:

```bash
kubectl create rolebinding default-admin --clusterrole=admin --serviceaccount=argo:default -n argo
```

**This grants admin privileges to the `default` `ServiceAccount` in the `argo` namespace.
Only Workflows in the `argo` namespace that use the `default` `ServiceAccount` receive these privileges.**
