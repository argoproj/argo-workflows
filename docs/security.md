# Security

[To report security issues](https://github.com/argoproj/argo-workflows/blob/main/SECURITY.md).

💡 Read [Practical Argo Workflows Hardening](https://blog.argoproj.io/practical-argo-workflows-hardening-dd8429acc1ce).

## Workflow Controller Security

This has three parts.

### Controller Permissions

The controller has permission (via Kubernetes RBAC + its config map) with either all namespaces (cluster-scope install) or a single [managed namespace](managed-namespace.md) (namespace-install), notably:

* List/get/update workflows, and cron-workflows.
* Create/get/delete pods, PVCs, and PDBs.
* List/get template, config maps, service accounts, and secrets.

See [`workflow-controller-cluster-role.yaml`](https://raw.githubusercontent.com/argoproj/argo-workflows/main/manifests/cluster-install/workflow-controller-rbac/workflow-controller-clusterrole.yaml) or [`workflow-controller-role.yaml`](https://raw.githubusercontent.com/argoproj/argo-workflows/main/manifests/namespace-install/workflow-controller-rbac/workflow-controller-role.yaml)

### User Permissions

Users minimally need permission to create/read workflows. The controller will then create workflow pods (config maps etc) on behalf of the users, even if the user does not have permission to do this themselves. The controller will only create workflow pods in the workflow's namespace.

A way to think of this is that, if the user has permission to create a workflow in a namespace, then it is OK to create pods or anything else for them in that namespace.

If the user only has permission to create workflows, then they will be typically unable to configure other necessary resources such as config maps, or view the outcome of their workflow. This is useful when the user is a service.

!!! Warning
    If you allow users to create workflows in the controller's namespace (typically `argo`), it may be possible for users to modify the controller itself.  In a namespace-install the managed namespace should therefore not be the controller's namespace.

You can typically further restrict what a user can do to just being able to submit workflows from templates using [the workflow restrictions feature](workflow-restrictions.md).

#### UI Access

If you want a user to have read-only access to the entirety of the Argo UI for their namespace, a sample role for them may look like:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: ui-user-read-only
rules:
  # k8s standard APIs
  - apiGroups:
      - ""
    resources:
      - events
      - pods
      - pods/log
    verbs:
      - get
      - list
      - watch
  # Argo APIs. See also https://github.com/argoproj/argo-workflows/blob/main/manifests/cluster-install/workflow-controller-rbac/workflow-aggregate-roles.yaml#L4
  - apiGroups:
      - argoproj.io
    resources:
      - eventsources
      - sensors
      - workflows
      - workfloweventbindings
      - workflowtemplates
      - clusterworkflowtemplates
      - cronworkflows
      - workflowtaskresults
    verbs:
      - get
      - list
      - watch
```

### Workflow Pod Permissions

Workflow pods run using either:

* The `default` service account.
* The service account declared in the workflow spec.

There is no restriction on which service account in a namespace may be used.

This service account typically needs [permissions](workflow-rbac.md).

Different service accounts should be used if a workflow pod needs to have elevated permissions, e.g. to create other resources.

The main container will have the service account token mounted, allowing the main container to patch pods (among other permissions). Set `automountServiceAccountToken` to false to prevent this. See [fields](fields.md).

By default, workflows pods run as `root`. To further secure workflow pods, set the [workflow pod security context](workflow-pod-security-context.md).

You should configure the controller with the correct [workflow executor](workflow-executors.md) for your trade off between security and scalability.

These settings can be set by default using [workflow defaults](default-workflow-specs.md).

## Argo Server Security

Argo Server implements security in three layers.

Firstly, you should enable [transport layer security](tls.md) to ensure your data cannot be read in transit.

Secondly, you should enable an [authentication mode](argo-server.md#auth-mode) to ensure that you do not run workflows from unknown users.

Finally, you should configure the `argo-server` role and role binding with the correct permissions.

### Read-Only

You can achieve this by configuring the `argo-server` role ([example](https://github.com/argoproj/argo-workflows/blob/main/manifests/namespace-install/argo-server-rbac/argo-server-role.yaml) with only read access (i.e. only `get`/`list`/`watch` verbs)).

## Network Security

Argo Workflows requires various levels of network access depending on configuration and the features enabled. The following describes the different workflow components and their network access needs, to help provide guidance on how to configure the argo namespace in a secure manner (e.g. `NetworkPolicy`).

### Argo Server

The Argo Server is commonly exposed to end-users to provide users with a UI for visualizing and managing their workflows. It must also be exposed if leveraging [webhooks](webhooks.md) to trigger workflows. Both of these use cases require that the argo-server Service to be exposed for ingress traffic (e.g. with an Ingress object or load balancer). Note that the Argo UI is also available to be accessed by running the server locally (i.e. `argo server`) using local KUBECONFIG credentials, and visiting the UI over <https://localhost:2746>.

The Argo Server additionally has a feature to allow downloading of artifacts through the UI. This feature requires that the argo-server be given egress access to the underlying artifact provider (e.g. S3, GCS, MinIO, Artifactory, Azure Blob Storage) in order to download and stream the artifact.

### Workflow Controller

The workflow-controller Deployment exposes a Prometheus metrics endpoint (workflow-controller-metrics:9090) so that a Prometheus server can periodically scrape for controller level metrics. Since Prometheus is typically running in a separate namespace, the argo namespace should be configured to allow cross-namespace ingress access to the workflow-controller-metrics Service.

### Database access

A persistent store can be configured for either [archiving](workflow-archive.md) or [offloading](offloading-large-workflows.md) workflows. If either of these features are enabled, both the workflow-controller and argo-server Deployments will need egress network access to the external database used for archiving/offloading.
