# Multi-Cluster

You can execute workflows that runs some or all of its pods in clusters other than the cluster the controller is installed in.

## Considerations 
### Limitations

Do not use this feature as a way to have single Argo Workflows installation managing many clusters as that creates a single-point of failure, also it cannot scale horizontally. 

This mode only listens to workflows within the controller's cluster. You cannot create workflows in other clusters - you must install a workflow controller in every cluster you create a workflow in.

This feature is orthogonal to managed namespaces. If you install in namespace-mode but configure multiple clusters - you can only run manage workflow pods in the namespace with exactly the same name, regardless of cluster.

### Networking

As we need to communicate cross-cluster - you'll be connecting across security groups. Consider how you set this up. This may not be allow in some organisations. 

### Security

You'll may want to run in a cluster-wide installation. As a result you'll end up with a controller than can create pods in any cluster it is aware of.

Use the principal of least-privilege when configuring your RBAC.

This may not be allow in some organisations.

### User Interface

The user interface is not multi-cluster aware (yet). You'll not be able to see the logs of your pods.

## Usage

To make the workflow controller aware of other clusters and able to connect to them:

```bash
kubectl -n argo create secret generic clusters
```

This need to be populate with one entry per cluster, e.g.:

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

To manually configure the base, take the following example JSON, enter your values, and base-64 encode it:

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

Another option. Download the KUBECONFIG into your local `~/.kube/config` and add it as follows:

```bash
argo cluster add my-other-cluster-name my-context-name 
```

Restart the workflow controller.

Much like you already do for the controller's cluster, create any service accounts, roles and role bindings you need to run workflow pods in your other cluster. E.g.

* [workflow-role.yaml](manifests/quick-start/base/workflow-role.yaml)
* [workflow-default-rolebinding.yaml](manifests/quick-start/base/workflow-default-rolebinding.yaml)

If you're using artifacts, e.g. you have a default artifact repository configured, create any secrets you need for it. 

## Setting A Cluster For All Workflows By  Default

Workflows run by default in the same cluster as the controller. You can change this by changing [default workflow spec](default-workflow-specs.md)