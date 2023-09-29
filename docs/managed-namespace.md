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

!!! Info "Example Use Case"
    You can use a managed namespace install if you want some users or services to run Workflows without granting them privileges in the namespace where Argo Workflows is installed.
    For example, if you only run CI/CD Workflows that are maintained by the same team that manages the Argo Workflows installation, you may want a namespace install.
    But if all the Workflows are run by a separate data science team, you may want to give them a "data-science-workflows" namespace and use a managed namespace install of Argo Workflows in another namespace.
