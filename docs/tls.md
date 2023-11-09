# Transport Layer Security

> v2.8 and after

If you're running Argo Server you have three options with increasing transport security (note - you should also be
running [authentication](argo-server.md#auth-mode)):

## Default configuration

> v2.8 - 2.12

Defaults to [Plain Text](#plain-text)

> v3.0 and after

Defaults to [Encrypted](#encrypted) if cert is available

Argo image/deployment defaults to [Encrypted](#encrypted) with a self-signed certificate which expires after 365 days.

## Plain Text

Recommended for: development.

Everything is sent in plain text.

Start Argo Server with the --secure=false (or `ARGO_SECURE=false`) flag, e.g.:

```bash
export ARGO_SECURE=false
argo server --secure=false
```

To secure the UI you may front it with a HTTPS proxy.

## Encrypted

Recommended for: development and test environments.

You can encrypt connections without any real effort.

Start Argo Server with the `--secure` flag, e.g.:

```bash
argo server --secure
```

It will start with a self-signed certificate that expires after 365 days.

Run the CLI with `--secure` (or `ARGO_SECURE=true`) and `--insecure-skip-verify` (or `ARGO_INSECURE_SKIP_VERIFY=true`).

```bash
argo --secure --insecure-skip-verify list
```

```bash
export ARGO_SECURE=true
export ARGO_INSECURE_SKIP_VERIFY=true
argo --secure --insecure-skip-verify list
```

Tip: Don't forget to update your readiness probe to use HTTPS. To do so, edit your `argo-server`
Deployment's `readinessProbe` spec:

```yaml
readinessProbe:
    httpGet: 
        scheme: HTTPS
```

### Encrypted and Verified

Recommended for: production environments.

Run your HTTPS proxy in front of the Argo Server. You'll need to set-up your certificates (this is out of scope of this
documentation).

Start Argo Server with the `--secure` flag, e.g.:

```bash
argo server --secure
```

As before, it will start with a self-signed certificate that expires after 365 days.

Run the CLI with `--secure` (or `ARGO_SECURE=true`) only.

```bash
argo --secure list
```

```bash
export ARGO_SECURE=true
argo list
```

### TLS Min Version

Set `TLS_MIN_VERSION` to be the minimum TLS version to use. This is v1.2 by default.

This must be one of these [int values](https://golang.org/pkg/crypto/tls/).

| Version | Value |
|---|---|
| v1.0 | 769 |
| v1.1 | 770 |
| v1.2 | 771 |
| v1.3 | 772 |
