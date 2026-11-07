Description: add support for secret ref in sso issuer and redirect URL configuration
Authors: [Gaurang](https://github.com/gaurang9991)
Component: General
Issues: 17082

You can now read `sso.issuer` and `sso.redirectUrl` from a Kubernetes Secret, via `sso.issuerSecret` and `sso.redirectUrlSecret`, instead of only setting them as literal strings in the ConfigMap.
This is useful when an OIDC operator (for example, Dex) writes the issuer URL into the same Secret as the client ID and client secret.
If both the literal and the Secret reference are set for a field, the literal value takes precedence.
