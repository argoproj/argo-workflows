# Multi-Cluster

You can run workflows where one or more tasks are run in another cluster or namespace.

## How It Works

You configure Argo to be aware of other clusters by adding secrets which contain KUBECONFIG.

Argo can be configured to watch for workflows and pods either for the whole cluster, or for a specific managed
namespace. With multi-cluster:

* The workflow controller will listen to what ever it has been configured to listen to.

## Configuration

For each namespace you want to have multi-cluster workflows in, we need to have a service account to represent the
namespace (so you can configure that service account's permissions) and the `argo` service needs to be able to
impersonate that account.

In the Argo system namespace (typically `argo`):

* Create a cluster secret using `argo cluster add ${clusterName}`.
* Create a service account with the same name as the namespace.
* Create a role and role binding that allows that service account to access the cluster secrets. Cluster secrets are
  named `cluster-${clusterName}`.
* Add the namespace to the `impersonator` in the `resources` section.

In the remote namespace:

* Create any secrets (e.g. to archive logs or artifacts) you need
* If your workflow uses a non-default service account, create that.
* Make sure that service account has a role binding to the standard workflow role.