# Links

> v2.7 and after

You can configure Argo Server to show custom links:

* A "Get Help" button in the bottom right of the window linking to you organization help pages or chat room.
* Deep-links to your facilities (e.g. logging facility) in the UI for both the workflow and each workflow pod.
* Adds a button to the top of workflow view to navigate to customized views.

Links can contain placeholder variables. Placeholder variables are indicated by the dollar sign and curly braces: `${variable}`.

These are the commonly used variables:

* `${metadata.namespace}`: Kubernetes namespace of the current workflow / pod / event source / sensor
* `${metadata.name}`: Name of the current workflow / pod / event source / sensor
* `${status.startedAt}`: Start time-stamp of the workflow / pod, in the format of `2021-01-01T10:35:56Z`
* `${status.finishedAt}`: End time-stamp of the workflow / pod, in the format of  `2021-01-01T10:35:56Z`. If the workflow/pod is still running, this variable will be `null`

See [workflow-controller-configmap.yaml](workflow-controller-configmap.yaml) for a complete example

> v3.1 and after

Epoch time-stamps are available now. These are useful if we want to add links to logging facilities like [Grafana](https://grafana.com/)
or [DataDog](https://datadoghq.com/), as they support Unix epoch time-stamp formats as URL
parameters:

* `${status.startedAtEpoch}`: Start time-stamp of the workflow/pod, in the Unix epoch time format in **milliseconds**, e.g. `1609497000000`.
* `${status.finishedAtEpoch}`: End time-stamp of the workflow/pod, in the Unix epoch time format in  **milliseconds**, e.g. `1609497000000`. If the workflow/pod is still running, this variable will represent the current time.

> v3.1 and after

In addition to the above variables, we can now access all [workflow fields](fields.md#workflow) under `${workflow}`.

For example, one may find it useful to define a custom label in the workflow and access it by `${workflow.metadata.labels.custom_label_name}`

We can also access workflow fields in a pod link. For example, `${workflow.metadata.name}` returns
the name of the workflow instead of the name of the pod. If the field doesn't exist on the workflow then the value will be an empty string.

> v3.6 and after

Links can be relative when configured and the UI will prefix them with the base UI URL.
