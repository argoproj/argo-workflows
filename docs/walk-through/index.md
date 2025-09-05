# About

Argo Workflows is implemented as a Kubernetes CRD (Custom Resource Definition) which are defined in YAML files. As a
result, Argo `workflows` can be managed using `kubectl` and natively integrate with other Kubernetes services such as
volumes, secrets, and RBAC. The Argo Workflows software is light-weight and installs in under a minute, and provides
complete workflow features including parameter substitution, artifacts, fixtures, loops and recursive workflows.

Dozens of examples are available in
the [`examples` directory](https://github.com/argoproj/argo-workflows/tree/main/examples) on GitHub.

For a complete description of the Argo workflow spec, please refer
to [the spec documentation](../fields.md#workflowspec).

Progress through the walk through in sequence to learn all the basics.

Start with [Argo CLI](argo-cli.md).

## Hera SDK for Python Users

If you would prefer to write Workflows using Python instead of YAML, check out the walk through for
[Hera, the Python SDK for Argo Workflows](https://hera.readthedocs.io/en/stable/walk-through/quick-start/).
