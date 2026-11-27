Description: Accept tokens issued by the SSO OIDC provider as bearer tokens.
Authors: [Bertrand Paquet](https://github.com/bpaquet)
Component: General
Issues: 12049

You can let clients call the Argo Server API with a token issued by your OIDC provider, without the browser login and without access to the Kubernetes API.
Set `sso.bearerToken.enabled: true` and send the token as `Authorization: Bearer <token>`.
The Argo Server verifies the token's signature, issuer, audience and expiry, then applies SSO RBAC and namespace delegation to its claims.
By default it accepts tokens whose audience includes the SSO client ID; use `sso.bearerToken.allowedAudiences` to change that.
If verification fails, the request is rejected: the token isn't tried with the `client` auth mode.
