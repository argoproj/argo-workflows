# Argo Server Auth Mode Token

![alpha](assets/alpha.svg)

> v2.10 and after

Token based authentication is typically necessary when integrating automation. 

To enable this, start the Argo Server with `--auth-mode token`.

A token authenticated request must have a `Authorization: Bearer token:PXnB9D6CGvFYzAFa0WeOj97Ik6uLEHXq` authorisation header.

Argo Server will look for the token it the secret in named `argo-server-tokens` in its namespace (often `argo`).

Tokens must be 40 characters long. They must not be guessable - otherwise your security is compromised.
 
Generate a token using:

```shell script
docker run --rm docker/whalesay sh -c "cat /dev/random | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1"
```

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: argo-server-tokens
stringData:
  PXnB9D6CGvFYzAFa0WeOj97Ik6uLEHXq: automation
``` 

The value of the token is a service account, e.g.:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: automation
```

As this is an automation, this service account would typically have limited access.