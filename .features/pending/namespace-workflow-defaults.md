Description: Set default Workflow values per namespace with a labelled ConfigMap
Authors: [Omkar Shendge](https://github.com/omkar619-dev)
Component: General
Issues: 14057

Workflow defaults could previously only be set once per controller, in the `workflowDefaults` key of the workflow controller ConfigMap.
A namespace can now supply its own defaults in a ConfigMap labelled `workflows.argoproj.io/configmap-type: WorkflowDefaults`.
The ConfigMap is discovered by that label rather than by a fixed name, so it can be called anything.
The value under its `workflowDefaults` key has the same shape as the controller-level one, so the same document works in either place.

Defaults are merged rather than replaced, and the more specific value wins: a value set on the Workflow beats the namespace defaults, which beat the controller defaults.

A namespace must have at most one ConfigMap with that label.
More than one is an error for every Workflow in that namespace, because there is no sensible way to choose between them.

A namespace without a labelled ConfigMap simply has no namespace-level defaults.
A ConfigMap that exists but is missing the `workflowDefaults` key, whose value is not valid YAML, or which sets a field that is not recognised, is an error rather than being ignored, so that defaults never silently fail to apply.
