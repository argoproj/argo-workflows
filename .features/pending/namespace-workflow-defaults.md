Description: Set default Workflow values per namespace with a labelled ConfigMap
Authors: [Omkar Shendge](https://github.com/omkar619-dev)
Component: General
Issues: 14057

Workflow defaults could previously only be set once per controller, in the `workflowDefaults` key of the workflow controller ConfigMap.
A namespace can now supply its own defaults in a ConfigMap labelled `workflows.argoproj.io/configmap-type: WorkflowDefaults`.
The controller finds the ConfigMap by that label rather than by a fixed name, so you can call it anything.
The value under its `workflowDefaults` key has the same shape as the controller-level one, so the same document works in either place.

The controller merges the defaults rather than replacing them, and the more specific value wins: a value set on the Workflow beats the namespace defaults, which beat the controller defaults.

`templateReferencing: Strict` doesn't restrict namespace defaults, because it sanitizes only the submitted Workflow spec.
A namespace ConfigMap can therefore set fields a user cannot set directly, such as `serviceAccountName`, `hostNetwork` or `podSpecPatch`, wherever the Workflow and its `WorkflowTemplate` leave them unset.
Write access to that ConfigMap carries the same trust as the controller-level `workflowDefaults`.

A namespace must have at most one ConfigMap with that label.
More than one is an error for every Workflow in that namespace, because there is no sensible way to choose between them.

A namespace without a labelled ConfigMap has no namespace-level defaults.
The controller treats a ConfigMap that exists but is missing the `workflowDefaults` key, whose value is not valid YAML, or which sets a field that is not recognized, as an error rather than ignoring it, so that defaults never silently fail to apply.
This also affects Workflows that are already running, not only new ones.
The controller reads the defaults again on every reconcile, so a broken ConfigMap, or a second labelled one, puts every running Workflow in the namespace into `Error`.
Under `templateReferencing: Secure` even a valid edit does this.
On every reconcile the controller re-merges the spec of a running `workflowTemplateRef` Workflow and compares it with the stored one, so any edit to the `spec` defaults that changes the result, including adding one, puts it into `Error`.
