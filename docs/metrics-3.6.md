# Metrics upgrade notes

Metrics have changed between 3.5 and 3.6.

You can now retrieve metrics using the OpenTelemetry Protocol using the [OpenTelemetry collector](https://opentelemetry.io/docs/collector/), and this is the recommended mechanism.

These notes explain the differences in using the prometheus `/metrics` endpoint to scrape metrics for a minimal effort upgrade. It is not recommended you follow this guide blindly, the new metrics have been introduced because they add value, and so they should be worth collecting and using.

## TLS

The prometheus `/metrics` endpoint defaults to TLS on.

To disable this set `.metricsConfig.secure` to `false`.

## New metrics

The following are new metrics:

* `build_info`
* `total_count`
* `pods_total_count`
* `controller_build_info`
* `cronworkflows_triggered_total`
* `workflowtemplate_triggered_total`
* `workflowtemplate_runtime`
* `k8s_request_duration`
* `queue_duration`
* `queue_longest_running`
* `queue_retries`
* `queue_unfinished_work`
* `pod_pending`

and can be disabled with

```yaml
metricsConfig:
  options:
    build_info:
      disable: true
    total_count:
      disable: true
    pods_total_count:
      disable: true
    controller_build_info:
      disable: true
    cronworkflows_triggered_total:
      disable: true
    workflowtemplate_triggered_total:
      disable: true
    workflowtemplate_runtime:
      disable: true
    k8s_request_duration:
      disable: true
    queue_duration:
      disable: true
    queue_longest_running:
      disable: true
    queue_retries:
      disable: true}
    queue_unfinished_work:
      disable: true
    pod_pending:
      disable: true
```

## Renamed metrics

If you are using these metrics in your recording rules, dashboards or alerts you will need to use their new name after the upgrade:

| Old name                           | New name                           |
|------------------------------------|------------------------------------|
| `argo_workflows_count`             | `argo_workflows_gauge`             |
| `argo_workflows_pods_count`        | `argo_workflows_pods_gauge`        |
| `argo_workflows_queue_depth_count` | `argo_workflows_queue_depth_gauge` |
| `log_messages`                     | `argo_workflows_log_messages`      |

## Custom metrics

Custom metric names and labels must be valid prometheus and OpenTelemetry names now. This prevents the use of `:`, which was usable in earlier versions of workflows

Custom metrics, as defined by a workflow, could be defined as one type (say counter) in one workflow, and then as a histogram of the same name in a different workflow. This would work in 3.5 if the first usage of the metric had reached TTL and been deleted. This will no-longer work in 3.6, and custom metrics may not be redefined. It doesn't really make sense to change a metric in this way, and the OpenTelemetry SDK prevents you from doing so.
