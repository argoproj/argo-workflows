# Managed Namespace

> v2.5 and after

You can install Argo in either cluster scoped or namespace scope configurations.
This dictates if you must set-up cluster roles or normal roles.

In namespace scope configuration, you must run both the Workflow Controller and
Argo Server using `--namespaced`. If you would like to have the workflows running in a separate
namespace, add `--managed-namespace` as well. (In cluster scope installation, don't include `--namespaced`
or `--managed-namespace`.)

For example:

```yaml
      - args:
        - --configmap
        - workflow-controller-configmap
        - --executor-image
        - argoproj/workflow-controller:v2.5.1
        - --namespaced
        - --managed-namespace
        - default
```

Please mind that both cluster scoped and namespace scoped configurations require "admin" role because some custom resource (CRD) must be created (and CRD is always a cluster level object)
