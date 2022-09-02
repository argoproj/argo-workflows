# Widgets

> v3.0 and after

Widgets are intended to be embedded into other applications using inline frames (`iframe`). This may not work with your configuration. You may need to:

* Run the Argo Server with an account that can read workflows. That can be done using `--auth-mode=server` and configuring the `argo-server` service account.
* Run the Argo Server with `--x-frame-options=SAMEORIGIN` or `--x-frame-options=`.
