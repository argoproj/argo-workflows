# Argo Server

> v2.5 and after

!!! Warning "HTTP vs HTTPS"
    Since v3.0 the Argo Server listens for HTTPS requests, rather than HTTP.

The Argo Server is a server that exposes an API and UI for workflows. You'll need to use this if you want to [offload large workflows](offloading-large-workflows.md) or the [workflow archive](workflow-archive.md).

You can run this in either "hosted" or "local" mode.

It replaces the Argo UI.

## Hosted Mode

Use this mode if:

* You want a drop-in replacement for the Argo UI.
* If you need to prevent users from directly accessing the database.

Hosted mode is provided as part of the standard [manifests](https://github.com/argoproj/argo-workflows/blob/main/manifests), [specifically in `argo-server-deployment.yaml`](https://github.com/argoproj/argo-workflows/blob/main/manifests/base/argo-server/argo-server-deployment.yaml) .

## Local Mode

Use this mode if:

* You want something that does not require complex set-up.
* You do not need to run a database.

To run locally:

```bash
argo server
```

This will start a server on port 2746 which you [can view](https://localhost:2746).

## Options

### Auth Mode

See [auth](argo-server-auth-mode.md).

### Managed Namespace

See [managed namespace](managed-namespace.md).

### Base HREF

If the server is running behind reverse proxy with a sub-path different from `/` (for example,
`/argo`), you can set an alternative sub-path with the `--base-href` flag or the `ARGO_BASE_HREF`
environment variable.

You probably now should [read how to set-up an ingress](#ingress)

### Transport Layer Security

See [TLS](tls.md).

### SSO

See [SSO](argo-server-sso.md). See [here](argo-server-sso-argocd.md) about sharing Argo CD's Dex with Argo Workflows.

## Access the Argo Workflows UI

By default, the Argo UI service is not exposed with an external IP. To access the UI, use one of the
following:

### `kubectl port-forward`

```bash
kubectl -n argo port-forward svc/argo-server 2746:2746
```

Then visit: <https://localhost:2746>

### Expose a `LoadBalancer`

Update the service to be of type `LoadBalancer`.

```bash
kubectl patch svc argo-server -n argo -p '{"spec": {"type": "LoadBalancer"}}'
```

Then wait for the external IP to be made available:

```bash
kubectl get svc argo-server -n argo
```

```bash
NAME          TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)          AGE
argo-server   LoadBalancer   10.43.43.130   172.18.0.2    2746:30008/TCP   18h
```

### Ingress

You can get ingress working as follows:

Add `ARGO_BASE_HREF` as environment variable to `deployment/argo-server`. Do not forget to add a trailing '/' character.

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: argo-server
spec:
  selector:
    matchLabels:
      app: argo-server
  template:
    metadata:
      labels:
        app: argo-server
    spec:
      containers:
      - args:
        - server
        env:
          - name: ARGO_BASE_HREF
            value: /argo/
        image: argoproj/argocli:latest
        name: argo-server
...
```

Create a ingress, with the annotation `ingress.kubernetes.io/rewrite-target: /`:

>If TLS is enabled (default in v3.0 and after), the ingress controller must be told
>that the backend uses HTTPS. The method depends on the ingress controller, e.g.
>Traefik expects an `ingress.kubernetes.io/protocol` annotation, while `ingress-nginx`
>uses `nginx.ingress.kubernetes.io/backend-protocol`

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: argo-server
  annotations:
    ingress.kubernetes.io/rewrite-target: /$2
    ingress.kubernetes.io/protocol: https # Traefik
    nginx.ingress.kubernetes.io/backend-protocol: https # ingress-nginx
spec:
  rules:
  - http:
      paths:
      - path: /argo(/|$)(.*)
        pathType: Prefix
        backend:
          service:
            name: argo-server
            port:
              number: 2746
```

[Learn more](https://github.com/argoproj/argo-workflows/issues/3080)

## Security

Users should consider the following in their set-up of the Argo Server:

### API Authentication Rate Limiting

Argo Server does not perform authentication directly. It delegates this to either the Kubernetes API Server (when `--auth-mode=client`) and the OAuth provider (when `--auth-mode=sso`). In each case, it is recommended that the delegate implements any authentication rate limiting you need.

### IP Address Logging

Argo Server does not log the IP addresses of API requests. We recommend you put the Argo Server behind a load balancer, and that load balancer is configured to log the IP addresses of requests that return authentication or authorization errors.

### Rate Limiting

> v3.4 and after

Argo Server by default rate limits to 1000 per IP per minute, you can configure it through `--api-rate-limit`. You can access additional information through the following headers.

* `X-Rate-Limit-Limit` - the rate limit ceiling that is applicable for the current request.
* `X-Rate-Limit-Remaining` - the number of requests left for the current rate-limit window.
* `X-Rate-Limit-Reset` - the time at which the rate limit resets, specified in UTC time.
* `Retry-After` - indicate when a client should retry requests (when the rate limit expires), in UTC time.
