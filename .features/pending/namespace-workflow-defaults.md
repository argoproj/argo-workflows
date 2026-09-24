Description: Set default Workflow values per namespace with a workflow-defaults ConfigMap
Authors: [Omkar Shendge](https://github.com/omkar619-dev)
Component: General
Issues: 14057

Workflow defaults could previously only be set once per controller, in the `workflowDefaults` key of the workflow controller ConfigMap.
A namespace can now supply its own defaults in a ConfigMap named `workflow-defaults` in that namespace.
The value under its `workflowDefaults` key has the same shape as the controller-level one, so the same document works in either place.

Defaults are merged rather than replaced, and the more specific value wins: a value set on the Workflow beats the namespace defaults, which beat the controller defaults.

The ConfigMap name is fixed rather than discovered by label, so a namespace cannot have more than one: Kubernetes already guarantees that names are unique within a namespace.

A namespace without the ConfigMap simply has no namespace-level defaults.
A ConfigMap that exists but is missing the `workflowDefaults` key, or whose value is not valid YAML, is an error rather than being ignored, so that defaults never silently fail to apply.
