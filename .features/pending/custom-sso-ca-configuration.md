<!-- Required: All of these fields are required, including at least one issue -->
Description: Allow custom CA certificate configuration for SSO OIDC provider connections
Authors: [bradfordwagner](https://github.com/bradfordwagner)
Component: Server
Issues: 7198

<!--
This feature adds support for custom TLS configuration when connecting to OIDC providers for SSO authentication.
This is particularly useful when your OIDC provider uses self-signed certificates or custom Certificate Authorities (CAs).

* Use this feature when your OIDC provider uses custom self-signed CA certificates
* Configure custom CA certificates either inline or by file path

## Configuration Examples

### Inline PEM content
```yaml
sso:
  # Custom PEM encoded CA certificate file contents
  rootCA: |-
    -----BEGIN CERTIFICATE-----
    MIIDXTCCAkWgAwIBAgIJAKoK/heBjcOuMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
    ...
    -----END CERTIFICATE-----
```

### File path reference
```yaml
sso:
  # Custom CA certificate file name
  rootCAFile: /etc/ssl/certs/custom-ca.pem
```

The system will automatically use certificates mounted to `/etc/ssl/certs` without additional configuration.
For production environments, always use proper CA certificates instead of skipping TLS verification.
-->
