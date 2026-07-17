Description: Execute resource templates with a shared resource agent
Authors: [Isitha Subasinghe](https://github.com/isubasinghe)
Component: General
Issues: 99999

Resource templates can now opt into agent-based execution by setting `resource.agent: true`.
Instead of creating a separate workflow pod for every resource-template node, a shared resource agent pod executes and monitors all opted-in resource templates for the workflow. This reduces pod creation overhead for workflows that manage many Kubernetes resources.

The resource agent runs under a dedicated `<workflow service account>-resource-agent` service account and requires permissions to create and watch the resource kinds used by the workflow. See `docs/resource-template.md` for configuration details, required RBAC, and examples.
