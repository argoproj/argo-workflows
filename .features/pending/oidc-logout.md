Description: Add OIDC provider logout support for Argo Server
Authors: [Darcy Cleaver](https://github.com/decleaver)
Component: General
Issues: 12389

Argo Server can redirect SSO users to the OIDC provider's discovered `end_session_endpoint` when they log out.
Provider logout is disabled by default, so existing SSO installations do not need to change their configuration when upgrading.
To enable provider logout, configure `sso.logoutRedirectUrl` with an absolute HTTP(S) URL.
Argo Server fails to start if the configured logout redirect URL is relative, contains a fragment, or is otherwise invalid.
Register the exact URL with the identity provider as an allowed post-logout redirect URI.
Provider logout is known to work with Keycloak 18 and later, which accepts the `client_id` and `post_logout_redirect_uri` parameters Argo Server sends.
Okta provider logout is not supported because its end-session endpoint requires `id_token_hint`; Argo Server does not retain the raw ID token because it can exceed browser cookie size limits.

Separately, Argo Server now normalizes `--base-href` values that omit the trailing slash.
For an SSO deployment using `--base-href /argo`, the automatically resolved OIDC redirect URI changes from the malformed `https://<host>/argooauth2/callback` to `https://<host>/argo/oauth2/callback`.
If your identity provider has that malformed URI registered as a workaround, register the corrected URI before upgrading.
