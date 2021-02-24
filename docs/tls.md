# Transport Layer Security

![GA](assets/ga.svg)

> v2.8 and after

If you're running Argo Server you have three options with increasing transport security (note - you should also be
running [authentication](argo-server.md#auth-mode)):

## Plain Text

*Recommended for: dev*

This is the default setting: everything is sent in plain text.

To secure the UI you may front it with a HTTPS proxy.

## Encrypted

*Recommended for: development and test environments*

You can encrypt connections without any real effort.

Start Argo Server with the `--secure` flag, e.g.:

```
argo server --secure
```

It will start with a self-signed certificate that expires after 365 days.

Run the CLI with `--secure` (or `ARGO_SECURE=true`) and `--insecure-skip-verify` (or `ARGO_INSECURE_SKIP_VERIFY=true`).

```
argo --secure --insecure-skip-verify list
```

```
export ARGO_SECURE=true
export ARGO_INSECURE_SKIP_VERIFY=true
argo --secure --insecure-skip-verify list
```

Tip: Don't forget to update your readiness probe to use HTTPS. To do so, edit your `argo-server`
Deployment's `readinessProbe` spec:

```
readinessProbe:
    httpGet: 
        scheme: HTTPS
```

### Encrypted and Verified

*Recommended for: production environments*

Run your HTTPS proxy in front of the Argo Server. You'll need to set-up your certificates and this out of scope of this
documentation.

Start Argo Server with the `--secure` flag, e.g.:

```
argo server --secure
```

As before, it will start with a self-signed certificate that expires after 365 days.

Run the CLI with `--secure` (or `ARGO_SECURE=true`) only.

```
argo --secure list
```

```
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

