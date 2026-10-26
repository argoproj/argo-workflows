# Header Authentication

> v4.2 and after

Header authentication lets a reverse proxy authenticate your users.
The proxy forwards each user's identity to Argo Server in HTTP headers.
Argo Server trusts those headers, so it never sees the user's credentials.

!!! Warning
    Anyone who can send a request with the right headers can act as any user.
    You must make sure that only your authenticating proxy can reach Argo Server.
    Read [Securing the proxy connection](#securing-the-proxy-connection) before you enable this mode.

## How it works

1. A user sends a request to your reverse proxy.
2. The proxy authenticates the user.
   The proxy removes or overwrites any identity headers that the client sent, so a client cannot choose its own identity.
3. The proxy adds its own identity headers and, if you configured one, the shared secret header.
4. Argo Server verifies the proxy and builds the user's claims from the headers.

## Enable header authentication

Start Argo Server with `--auth-mode=header`.
You can combine it with other modes, for example `--auth-mode=sso --auth-mode=header`.

Configure the headers in the `header` section of the workflow controller ConfigMap.
See the [ConfigMap reference](workflow-controller-configmap.yaml) for every field.

```yaml
data:
  header: |
    # Issuer claim, either from an HTTP header or a static value.
    iss:
      value: https://auth.example.com
    sub:
      header: X-Forwarded-User
    email:
      header: X-Forwarded-Email
    preferred_username:
      header: X-Forwarded-User
    groups:
      header: X-Forwarded-Groups
    sharedSecret:
      header: X-Proxy-Auth
      requiredValue:
        name: argo-server-header-auth
        key: proxy-auth
    rbac:
      enabled: true
```

The subject is required.
If the subject is missing or empty, Argo Server treats the request as having no header credentials.

Separate multiple groups with commas, for example `admin,developer`.
You can also send the groups header more than once.
Argo Server keeps whitespace, so do not add spaces after commas.

## Securing the proxy connection

Without protection, any client that can reach Argo Server can send the identity headers itself.
Use a shared secret, and restrict network access as well.

### Shared secret

Configure `sharedSecret` with a header name and a Kubernetes Secret key.
Configure your proxy to send that header, with the secret as its value, on every request.

Argo Server handles the secret in three ways:

* If the secret is correct, Argo Server accepts the identity headers.
* If the secret is wrong, Argo Server rejects the request with `401`.
* If the secret is missing, Argo Server ignores the identity headers and treats the request as having no header credentials.

Argo Server reads the secret at startup, so restart it after you rotate the secret.
Update the proxy and the Secret together, because requests with the old value fail until the restart completes.

### Trusting headers without authentication

The `--insecure-trust-unauthenticated-headers` flag turns off the shared secret requirement.
With it, anyone who can reach the server's port can impersonate any user.
Use it only when nothing except your proxy can reach Argo Server.
For example, run the proxy as a sidecar and apply a `NetworkPolicy` that blocks all other traffic to the server.
Argo Server listens on all interfaces, so a sidecar alone does not stop other pods from connecting.
Argo Server logs a warning at startup when you set the flag.

### Network hardening

Do not rely on one control alone.

* Apply a `NetworkPolicy` that allows only the proxy to reach Argo Server.
* Use TLS between the proxy and Argo Server.
* Prevent workflow pods from reaching Argo Server directly.
  A user who can run a workflow could otherwise send forged headers.

## Order of authentication

Argo Server checks credentials in this order and skips disabled modes:

1. The `Authorization` header (`sso` or `client`).
2. Trusted headers (`header`).
3. The `Authorization` cookie (`sso` or `client`).
4. `server` mode.

If a request has no credentials for a method, Argo Server moves to the next method.
An empty `Authorization` header counts as no credentials.
If a request has credentials that a method recognizes and they are invalid, authentication fails.
Invalid credentials never fall through to another method.

If an `Authorization` value matches no enabled mode, Argo Server runs the request as the server when `server` mode is enabled.
Otherwise, it rejects the request.

!!! Note
    A request that carries identity headers but no shared secret never becomes the claimed user.
    If `server` mode is enabled, it runs as the Argo Server service account.
    If `server` mode is disabled, Argo Server rejects it.
    To reject all requests without valid credentials, do not enable `server` or `hybrid`.

## RBAC

If you do not enable RBAC for header users, every header user runs as the Argo Server service account.
To give users different permissions, set `rbac.enabled: true` in the `header` configuration.

Header users match the same service accounts and `workflows.argoproj.io/rbac-rule` annotations as [SSO RBAC](argo-server-sso.md#sso-rbac).
If you enable both `sso` and `header`, a rule applies to users from either source.
To limit a rule to one source, match on the issuer:

```text
iss == "https://idp.example.com" && "admins" in groups
```

Set a distinct issuer for header users so that you can match on it.
Groups come from the proxy, so they are only as trustworthy as the proxy.

### Namespace delegation

Set `HEADER_DELEGATE_RBAC_TO_NAMESPACE=true` on Argo Server to let namespace owners define rules for header users.
It works like [SSO namespace delegation](argo-server-sso.md#sso-rbac-namespace-delegation).
It is independent of `SSO_DELEGATE_RBAC_TO_NAMESPACE`.
Delegation does not apply when you run with `--namespaced`.

## Troubleshooting

If you see `token not valid`, no enabled mode accepted the request.
Check the [FAQ](faq.md) for common causes.
