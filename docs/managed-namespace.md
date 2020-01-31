# Managed Namespace

![alpha](assets/alpha.svg)

> v2.5 and after

You can install Argo in either cluster scoped or namespace scope configurations. This dictates if you must set-up cluster roles or normal roles.

In namespace scope configuration, you must run both the Workflow Controller and Argo Server using `--namespaced`.