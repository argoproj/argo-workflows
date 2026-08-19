Description: Add OIDC provider logout support for Argo Server
Authors: [Darcy Cleaver](https://github.com/decleaver)
Component: General
Issues: 12389

Argo Server can redirect SSO users to the OIDC provider's discovered `end_session_endpoint` when they log out.
Provider logout is disabled by default, so existing SSO installations do not need to change their configuration when upgrading.
To enable provider logout, configure `sso.logoutRedirectUrl` with an absolute HTTP(S) URL.
Argo Server rejects relative or invalid values at startup.
Register the exact URL with the identity provider as an allowed post-logout redirect URI.
Argo Server does not send an `id_token_hint`, so the identity provider may show the user a logout confirmation prompt.
