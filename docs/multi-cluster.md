# Multi-Cluster

You can execute workflows where some pods run in clusters other than the cluster the controller is installed in.

## Considerations 

### Not Intended For A Single Control Plane

Do not use this feature as a way to have single Argo Workflows installation managing many clusters as that creates a single-point of failure, and it will not scale. 

This mode only listens to workflows within the controller's cluster. You cannot create workflows in other clusters - you must install a workflow controller in every cluster you create a workflow in.

### Typically, Needs Cluster Wide Install

This feature is orthogonal to managed namespaces. If you install in namespace-mode but configure multiple clusters - you can only run manage workflow pods in the namespace with exactly the same name, regardless of cluster.

### Networking

As we need to communicate cross-cluster - you'll be connecting across security groups. Consider how you set this up. 

This may not be allowed in some organisations. 

### Security

You'll may want to run in a cluster-wide installation. As a result you'll end up with a controller than can create pods in any cluster it is aware of.

Use the principal of least-privilege when configuring your RBAC.

This may not be allowed in some organisations.

## Usage

### Configured The Workflow Controller 

To make the workflow controller aware of other clusters and able to connect to them:

```bash
kubectl -n argo create secret generic clusters
```

This needs to be populated with one entry per cluster, e.g.:

```yaml
apiVersion: v1
data:
  other: eyJIb3N0Ijoi...
kind: Secret
metadata:
  name: clusters
  namespace: argo
type: Opaque
```

To manually configure a cluster, take the following example JSON, enter your values, and base-64 encode it:

```json
{
  "Host": "https://0.0.0.0:57667",
  "APIPath": "",
  "Username": "",
  "Password": "",
  "BearerToken": "*******",
  "TLSClientConfig": {
    "Insecure": false,
    "ServerName": "",
    "CertFile": "",
    "KeyFile": "",
    "CAFile": "",
    "CertData": null,
    "KeyData": null,
    "CAData": "******",
    "NextProtos": null
  },
  "UserAgent": "",
  "DisableCompression": false,
  "QPS": 0,
  "Burst": 0,
  "Timeout": 0
}
```

Alternatively, download the KUBECONFIG into your local `~/.kube/config` and add it as follows:

```bash
argo cluster add my-other-cluster-name my-context-name 
```

Next, we only run pods in cluster-namespaces that have been explicitly allowed by configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  namespaceRoles: |
    # This example declares which cluster-namespaces for workflows created in the "argo" 
    # namespace the controller is allowed to run workflow pods in.
    - namespace: argo
      rules:
        # You must include one entry for the default cluster if you wish to continue
        # running pods there.
        - clusterNames:
            - default
          namespaces:
            # Use "" for all namespaces, e.g. if you have a cluster-install.
            - argo            
        # List other clusters-namespaces that this namespace will be able to declare
        # workflow pods in.
        - clusterNames:
            - other
          namespaces:
            - other-ns
```

Finally, restart the workflow controller.

### Configure Your Other Cluster

Much like you already do for the controller's cluster, in the other cluster, create any service accounts, roles and role bindings you need to run workflow pods in your other cluster. E.g.

* [workflow-role.yaml](manifests/quick-start/base/workflow-role.yaml)
* [workflow-default-rolebinding.yaml](manifests/quick-start/base/workflow-default-rolebinding.yaml)

If you're using artifacts, e.g. you have a default artifact repository configured, create any secrets you need for it. 

### Run Your Multi-Cluster Workflow

Example:

```yaml
metadata:
  generateName: multi-cluster-
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
         - name: this
           template: this
         - name: other
           template: other
    - name: this
      container:
        image: argoproj/argosay:v2
    - name: other
      clusterName: other
      namespace: argo
      serviceAccount: workflow
      container:
        image: argoproj/argosay:v2
```