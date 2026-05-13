Description: Add optional PKCE (RFC 7636) support to OAuth2 SSO authorization code flow
Author: [Robert Marklund](https://github.com/euforia)
Component: General
Issues: 16152

Argo Server SSO now supports PKCE (RFC 7636) on the OAuth 2.0 authorization code flow, opt-in via `enablePKCEAuthentication: true` under the `sso` section of the workflow-controller configmap.
Enabling PKCE is required by some authorization servers and is recommended by the OAuth 2.0 Security Best Current Practice (RFC 9700) for all clients, including confidential server-side clients, because it mitigates authorization code injection attacks that are not prevented by `state` or `client_secret` alone.
Argo only uses the `S256` challenge method; the deprecated `plain` method is never used.
The default is unchanged for backward compatibility.
