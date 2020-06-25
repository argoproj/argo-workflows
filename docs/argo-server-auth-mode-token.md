# Argo Server Auth Mode Token

![alpha](assets/alpha.svg)

> v2.10 and after

Token based authentication is typically necessary when integrating automation. 

To enable this, start the Argo Server with `--auth-mode token`.

A token authenticated request must have a `Authorization: Bearer token:my-token` authorisation header.

Argo Server will look for the token it the secret in named `argo-server-tokens` in its namespace (often `argo`).

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: argo-server-tokens
stringData:
  my-token: automation
``` 

The value of the token is a service account, e.g.:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: automation
```

As this is an automation, this service account would typically have limited access.