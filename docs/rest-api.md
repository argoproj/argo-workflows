# REST API

## Argo Server API

![GA](assets/ga.svg)

> v2.5 and after

Argo Workflows ships with a server that provide more features and security than before.

The server can be configured with or without client auth (`server --auth-mode client`). When it is disabled, then clients must pass their Kubeconfig base 64 encoded in the HTTP `Authorization` header:

```
ARGO_TOKEN=$(argo auth token)
curl -H "Authorization: $ARGO_TOKEN" http://localhost:2746/api/v1/workflows/argo
```

* Learn more on [how to generate an access token](access-token.md).

You can view the API reference docs in the Argo Server UI: http://localhost:2746/apidocs or [open OpenAPI spec](https://github.com/argoproj/argo/blob/master/api/openapi-spec/swagger.json)

