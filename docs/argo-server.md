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

You probably now should [read how to set-up an ingress](#ingress)

### Transport Layer Security

See [TLS](tls.md).

### SSO 

See [SSO](argo-server-sso.md).


## Access the Argo Workflows UI

By default, the Argo UI service is not exposed with an external IP. To access the UI, use one of the
following:

### `kubectl port-forward`

```sh
kubectl -n argo port-forward svc/argo-server 2746:2746
```

Then visit: http://127.0.0.1:2746


### Expose a `LoadBalancer`

Update the service to be of type `LoadBalancer`.

```sh
kubectl patch svc argo-server -n argo -p '{"spec": {"type": "LoadBalancer"}}'
```

Then wait for the external IP to be made available:

```sh
kubectl get svc argo-server -n argo
```
```sh
NAME          TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)          AGE
argo-server   LoadBalancer   10.43.43.130   172.18.0.2    2746:30008/TCP   18h
```

### Ingress

You can get ingress working as follows:

Update `service/argo-server` spec with `type: LoadBalancer`:

```yaml
    - name: web
      port: 2746
      targetPort: 2746
  type: LoadBalancer
```

Add `BASE_HREF` as environment variable to `deployment/argo-server` :


```yaml
        - name: argo-server
          image: argoproj/argocli:latest
          args: [server]
          env:
            - name: BASE_HREF
              value: /argo/
```

Create a ingress, with the annotation `ingress.kubernetes.io/rewrite-target: /`:

```yaml
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: argo-server
  annotations:
    ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
    - http:
        paths:
          - backend:
              serviceName: argo-server
              servicePort: 2746
            path: /argo
```

[Learn more](https://github.com/argoproj/argo/issues/3080)
