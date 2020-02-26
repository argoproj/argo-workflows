# Prometheus Metrics

![alpha](assets/alpha.svg)

> v2.7 and after

## Introduction

Custom prometheus metrics can be defined to be emitted on a `Workflow`- and `Step/Task`-level basis. These can be useful
for many cases; some examples:

- Keeping track of the duration of a `Workflow` or `Step/Task` over time, and setting an alert if it goes beyond a threshold
- Keeping track of the number of times a `Workflow` or `Step/Task` fails over time
- Reporting an important internal metric, such as a model training score or an internal error rate

Emitting custom metrics with Argo is easy, but it's important to understand what makes a good Prometheus metric and the
best way to define metrics in Argo to avoid problems such as [cardinality explosion](https://stackoverflow.com/questions/46373442/how-dangerous-are-high-cardinality-labels-in-prometheus).

## Metrics and metrics in Argo

Emitting metrics is the responsibility of the emitter owner. Since the user defines Workflows in Argo, the user is responsible
for emitting metrics correctly.

### What is and isn't a Prometheus metric

Prometheus metrics should be thought of as ephemeral data points of running processes; i.e., they are the answer to
the question "What is the state of my system _right now_?". Metrics should report things such as:

- a counter of the number of times a workflow or steps has failed, or
- a gauge of workflow duration, or
- an average of an internal metric such as a model training score or error rate.

Metrics are then routinely scraped and stored and -- when they are correctly designed -- they can represent time series.
Aggregating the examples above over time could answer useful questions such as:

- How has the error rate of this workflow or step changed over time?
- How has the duration of this workflow changed over time? Is the current workflow running for too long?
- Is our model improving over time?

Prometheus metrics should **not** be thought of as a store of data. Since metrics should only report the state of the system
at the current time, they should not be used to report historical data such as:

- the status of an individual instance of a workflow, or
- how long a particular instance of a step took to run.

Metrics are also ephemeral, meaning there is no guarantee that they will be persisted for any amount of time. If you need
a way to view and analyze historical data, consider the [workflow archive](workflow-archive.md) or reporting to logs.


### Metric types

Please see the [Prometheus docs on metric types](https://prometheus.io/docs/concepts/metric_types/).

### How metrics work in Argo

In order to analyze the behavior of a workflow over time, we need to be able to link different instances
(i.e. individual executions) of a workflow together into a "series" for the purposes of emitting metrics. We do so by linking them together
with the same metric descriptor.

In prometheus, a metric descriptor is defined as a metric's name and its key-value labels. For example, for a metric 
tracking the duration of model execution over time, a metric descriptor could be:

`argo_workflows_model_exec_time{model_name="model_a",phase="validation"}`

 This metric then represents the amount of time that "Model A" took to train in the phase "Validation". It is important
 to understand that the metric name _and_ its labels form the descriptor: `argo_workflows_model_exec_time{model_name="model_b",phase="validation"}`
 is a different metric (and will track a different "series" altogether).
 
 Now, whenever we run our first workflow that validates "Model A" a metric with the amount of time it took it to do so will
 be created and emitted. For each subsequent time that this happens, no new metrics will be emitted and the _same_ metric
 will be updated with the new value. Since, in effect, we are interested on the execution time of "validation" of "Model A"
 over time, we are no longer interested in the previous metric and can assume it has already been scraped.
 
 In summary, whenever you want to track a particular metric over time, you should use the same metric name _and_ metric
 labels wherever it is emitted. This is how these metrics are "linked" as belonging to the same series.
 
 ## Define metrics in Argo
 
 In Argo you can define a metric on the `Workflow` level or on the `Step/Task` level.



