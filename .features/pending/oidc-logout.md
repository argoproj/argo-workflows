Description: Add OIDC provider logout support for Argo Server
Authors: [Darcy Cleaver](https://github.com/your-github-handle)
Component: General
Issues: 12389

Argo Server can redirect SSO users to the OIDC provider's discovered `end_session_endpoint` when they log out.
Configure `--logout-redirect-url` with an absolute URL registered with the identity provider as a post-logout redirect URI.