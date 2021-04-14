# Links

![GA](assets/ga.svg)

> v2.7 and after

You can configure Argo Server to show custom links:

* A "Get Help" button in the bottom right of the window linking to you organisation help pages or chat room. 
* Deep-links to your facilities (e.g. logging facility) in the user interface for both the workflow and each workflow pod.

Links can contain placeholder variables. Placeholder variables are indicated by the dollar sign and curly braces (`${variable}`).
There are currently 4 variables available:

| Variable                | Description                                                                                                                                               |
|:-----------------------:|-----------------------------------------------------------------------------------------------------------------------------------------------------------|
| `${metadata.namespace}` | Kubernetes namespace of the current workflow/pod/event source/sensor                                                                                      |
| `${metadata.name}`      | Name of the current workflow/pod/event source/sensor                                                                                                      |
| `${status.startedAt}`   | Start timestamp of the workflow/pod, in the format of `2021-01-01T10:35:56Z`                                                                              |
| `${status.finishedAt}`  | End timestamp of the workflow/pod, in the format of  `2021-01-01T10:35:56Z`. If the workflow/pod is still running, this variable will be an empty string. |

> v3.1 and after

In addition to the above variables, we can now access all [workflow fields](fields.md#workflow).
To access a workflow field, use `${workflow}`.

For example, one may find it useful to define a custom label in the workflow and access it by `${workflow.metadata.labels.custom_label_name}`

We can also access workflow fields with `${workflow}` in a pod link. For example, `${workflow.metadata.name}` returns
the name of the workflow instead of the name of the pod.

## Filters

> v3.1 and after

Link placeholder now supports filters powered by [LiquidJS](https://liquidjs.com/filters/overview.html).

For example, to convert the start timestamp to a different time format:

```
${status.startedAt | date: '%Y-%m-%d %H:%M'}
```

This is useful if we want to add links to logging facilities like [Grafana](https://grafana.com/)
or [DataDog](https://datadog.com/), as they support Unix Epoch timestamp in the URL
parameters.

For example, to link to a specific time range of a Grafana dashboard:

```yaml
links: |
  - name: Grafana
    scope: workflow
    url: "https://grafana/dashboard?workflowName=${metadata.name}&from=${status.startedAt | date: '%s'}&to=${ status.finishedAt | date: '%s' | default: 'now' }"
```

This will make sure `status.startedAt` and `status.finishedAt` are converted to Unix epoch time in seconds which is support by
Grafana. Furthermore, `status.finishedAt` will be converted to `now` if the workflow is still running.

!!! Warning
  Since the URL is provided through a YAML string. It's important to make sure the YAML format is valid. Double quote the URL if 
  it includes includes special characters, (e.g. `:`). Escape any internal double quotes into `\"` or use single quotes instead.

  For example: `url: "https://grafana/dashboard?to=${ status.finishedAt | date: '%s' | default: 'now' }"`

See [workflow-controller-configmap.yaml](workflow-controller-configmap.yaml) for a complete example
