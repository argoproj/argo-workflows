# Security

## Workflow Controller Security

This has three parts.

### Controller Permissions

The controller is has permission (via Kubernetes RBAC + its config map) with either all namespaces (cluster-scope install) or a single managed namespace (namespace-install):

* read/update workflows, and cron-workflows
* create/get/delete pods, PVCs, and PDBs
* read template, config maps, service accounts, and secrets

See [workflow-controller-clusterrole.yaml](manifests/cluster-install/workflow-controller-rbac/workflow-controller-clusterrole.yaml) on [workflow-controller-role.yaml](manifests/namespace-install/workflow-controller-rbac/workflow-controller-role.yaml)

### User Permissions

Users minimally need permission to create/read workflows. The controller will then create workflow pods (config maps etc) on behalf of the users, even if the user does not have permission to do this themselves, effectively allowing for privilege escalation. 

Another way to think of this is that, if the user has permission to create a workflow in a namespace, then it is OK to create pods or anything else for them in that namespace.

Implicitly, users should not be given permission to create workflows if they are not allowed to create pods etc.

The controller will only create workflow pods in:

* The workflow's cluster and namespace. 
* Another cluster or namespace that the controller has been explicitly configured to do so using `namespaceRoles` (advanced). 

If the user only has permission to create workflows, then they will be typically unable to configure other necessary resources such as config maps, or view the outcome of their workflow. This is useful when the user is a service.  

!!! Warning
    If you allow users to create workflows in the controller's namespace (typically `argo`), it may be possible for users to modify the controller itself.  In a namespace-install the managed namespace should therefore not be the controller's namespace.

### Workflow Pod Permissions

Finally, the workflows pods themselves run using either:

* The `default` service account.
* The service account declared in the workflow spec when it was created by the user. (in >= v3.0)

Since the controller assumes that if the workflow was created and therefore the user is allowed to utilize resources in that namespace, then there is no restriction on which service account may be used.

Different service accounts are useful if a workflow pod needs to have elevated permissions, e.g. to create other resources.

By default, workflows pods run as `root`. To further secure workflow pods, set the [workflow pod security context](workflow-pod-security-context.md).

Finally, with the `docker` executor, workflow pods run in "privileged" mode. Your choice of [workflow executor](workflow-executors.md) has a massive impact on security. The `k8sapi` executor is the most secure executor to use, `docker` the least. The `pns` executor is strong in both security and performance.

## Argo Server Security

Argo Server implements security in three layers.

Firstly, you should enable [transport layer security](tls.md) to ensure your data cannot be read in transit.

Secondly, you should enable an [authentication mode](argo-server.md#auth-mode) to ensure that you do not run workflows from unknown users.

Finally, you should configure the `argo-server` role and role binding with the correct permissions.

### Read-Only

You can achieve this by configuring the `argo-server` role ([example](https://github.com/argoproj/argo/blob/master/manifests/namespace-install/argo-server-rbac/argo-server-role.yaml) with only read access (i.e. only `get`/`list`/`watch` verbs).
