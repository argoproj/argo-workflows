# REST API

## Argo Server API

> v2.5 and after

Argo Workflows ships with a server that provides more features and security than before.

The server can be configured with or without client auth (`server --auth-mode client`). When it is disabled, then clients must pass their KUBECONFIG base 64 encoded in the HTTP `Authorization` header:

```bash
ARGO_TOKEN=$(argo auth token)
curl -H "Authorization: $ARGO_TOKEN" https://localhost:2746/api/v1/workflows/argo
```

* Learn more on [how to generate an access token](access-token.md).

API reference docs :

* [Latest docs](swagger.md) (maybe incorrect)
* Interactively in the [Argo Server UI](https://localhost:2746/apidocs). (>= v2.10)
