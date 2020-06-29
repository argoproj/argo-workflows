# Argo Server

![GA](assets/ga.svg)

> v2.5 and after

The Argo Server is a server that exposes an API and UI for workflows. You'll need to use this if you want to [offload large workflows](offloading-large-workflows.md) or the [workflow archive](workflow-archive.md).

You can run this in either "hosted" or "local" mode.

It replaces the Argo UI.

## Hosted Mode

Use this mode if:

* You want a drop-in replacement for the Argo UI.
* If you need to prevent users from directly accessing the database.

Hosted mode is provided as part of the standard [manifests](https://github.com/argoproj/argo/blob/master/manifests), [specifically in argo-server-deployment.yaml](https://github.com/argoproj/argo/blob/master/manifests/base/argo-server/argo-server-deployment.yaml) .

## Local Mode

Use this mode if:

* You want something that does not require complex set-up.
* You do not need to run a database.

To run locally:

```
argo server
```

This will start a server on port 2746 which you can view at [http://localhost:2746](http://localhost:2746).

## Options

### Auth Mode

See [auth](argo-server-auth-mode.md).

### Managed Namespace

See [managed namespace](managed-namespace.md).

### Base href

If the server is running behind reverse proxy with a subpath different from `/` (for example, 
`/argo`), you can set an alternative subpath with the `--base-href` flag or the `BASE_HREF` 
environment variable.

### Transport Layer Security

See [TLS](tls.md).

### SSO 

See [SSO](argo-server-sso.md).


## Access the Argo Workflows UI

```sh
kubectl -n argo port-forward deployment/argo-server 2746:2746
```

Then visit: http://127.0.0.1:2746

By default, the Argo UI service is not exposed with an external IP. To access the UI, use one of the
following:

### Method 1: kubectl port-forward

```
kubectl -n argo port-forward deployment/argo-server 2746:2746
```

Then visit: http://127.0.0.1:8001

### Method 2: kubectl proxy

```
kubectl proxy
```

Then visit: http://127.0.0.1:8001/api/v1/namespaces/argo/services/argo-ui/proxy/

NOTE: artifact download and webconsole is not supported using this method

### Method 3: Expose a LoadBalancer

Update the argo-ui service to be of type `LoadBalancer`.

```
kubectl patch svc argo-ui -n argo -p '{"spec": {"type": "LoadBalancer"}}'
```

Then wait for the external IP to be made available:

```
kubectl get svc argo-ui -n argo
NAME      TYPE           CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
argo-ui   LoadBalancer   10.19.255.205   35.197.49.167   80:30999/TCP   1m
```

NOTE: On Minikube, you won't get an external IP after updating the service -- it will always show
`pending`. Run the following command to determine the Argo UI URL:

```
minikube service -n argo --url argo-ui
```
