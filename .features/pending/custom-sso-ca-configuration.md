Description: Allow custom CA certificate configuration for SSO OIDC provider connections
Authors: [bradfordwagner](https://github.com/bradfordwagner)
Component: General
Issues: 7198

This feature adds support for custom TLS configuration when connecting to OIDC providers for SSO authentication.
This is particularly useful when your OIDC provider uses self-signed certificates or custom Certificate Authorities (CAs).

    - Use this feature when your OIDC provider uses custom self-signed CA certificates
    - Configure custom CA certificates either inline or by file path

**Configuration Examples**
**Inline PEM content**

    sso:
      # Custom PEM encoded CA certificate file contents
      rootCA: |-
        -----BEGIN CERTIFICATE-----
        MIIDXTCCAkWgAwIBAgIJAKoK/heBjcOuMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
        ...
        -----END CERTIFICATE-----

The system will automatically use certificates configured with SSL_CERT_DIR, and SSL_CERT_FILE for non macOS environments.
For production environments, always use proper CA certificates instead of skipping TLS verification.
