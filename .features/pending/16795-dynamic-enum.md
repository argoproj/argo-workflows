Description: Fill parameter dropdowns from live Kubernetes resources with enumFrom
Authors: [Lars W](https://github.com/lwarratz)
Component: UI
Issues: 16795

Parameters can now use `enumFrom` instead of `enum` to fill their UI dropdown from live Kubernetes resources, such as namespaces or Deployments.
`enumFrom` names a resource `apiVersion` and `kind`, an optional namespace and label selector, and a JSONPath that picks a value from each item.
The namespace and label selector can reference other parameters, so one dropdown can depend on another, for example Deployments in the chosen namespace.
The Argo Server resolves the values with the user's own Kubernetes credentials, and reads the `enumFrom` from the stored template rather than trusting the browser, so users only see resources they are allowed to list.
Secrets can never be listed.
In `server` auth mode, every user sees what the Argo Server's service account can list.
To protect the Kubernetes API, results are capped at 500 options and cached per user for 10 seconds; both are configurable.
If the values can't be resolved, the UI falls back to a text box.
`argo submit --from <kind/name> --interactive` asks for each parameter on the command line and lists the same options.
The submitted value is not checked against the options.
See [Dynamic Parameter Enums](dynamic-parameter-enums.md).
