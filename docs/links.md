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

See [workflow-controller-configmap.yaml](workflow-controller-configmap.yaml)

> v3.1 and after

Link placeholder now supports expression evaluation in JavaScript.
We have also provided a utility function `toEpoch` to convert `status.startedAt`
and `status.finishedAt` to Unix epoch time in seconds. This is especially useful if you
want to add a link to your logging facilities like [Grafana](https://grafana.com/) or [DataDog](https://datadog.com/),
as they support epoch timestamp in URL parameters.

For example, to link to a specific time range of a Grafana dashboard:

```yaml
links: |
  - name: Grafana
    scope: workflow
    url: https://grafana/dashboard?workflowName=${metadata.name}&from=${toEpoch(status.startedAt) * 1000}&to=${(toEpoch(status.finishedAt) * 1000) || "now"}
```

This will make sure `status.startedAt` and `status.finishedAt` are converted to Unix epoch time in milliseconds which is support by
Grafana. Furthermore, `status.finishedAt` will be evaluated as `now` if the workflow is still running.
