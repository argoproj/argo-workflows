# Events

> v2.11 and after

![alpha](assets/alpha.svg)

## Overview

To support external webhooks, we have this endpoint `/api/v1/events/{namespace}`. Events can be sent to that can be any JSON data.

These events can:

* [Submit from a workflow template](#submitting-from-a-workflow-template) (or cluster workflow template).
* [Resume a suspended workflow node](#resume-a-suspended-workflow-node).
* [Gate a cron workflow](#gate-a-cron-workflow)

In each use case, the resource must match the event based on an expression.

## Authentication and Security

Clients wanting to send events to the endpoint need an [access token](access-token.md).  The token may be namespace-scoped or cluster-scoped. If it is namespace scoped, the namespace must be sent in the URL, but if it is cluster-scoped, then the namespace can be the empty string.  

It is only possible to submit/resume resources your access token has access to. 

Example:

```bash
curl https://localhost:2746/api/v1/events/argo \
  -H "Authorization: $ARGO_TOKEN" \
  -d '{"type": "test"}'
```

Or cluster-scoped:

```bash
curl https://localhost:2746/api/v1/events/ \
  -H "Authorization: $ARGO_TOKEN" \
  -d '{"type": "test"}'
```

The event endpoint will always return in under 10 seconds because the event will be queued and processed asynchronously. This means you will not be notified synchronously of failure. It will only return a failure (503) if the queue is full, either due to a large number of events, or problems processing those events.  

!!! WARNING
    Events may not always be processed sequentially.   

## Event Expression and the Event Expression Environment

**Event expressions** are expressions that are evaluated over the **event expression environment**.

The event environment typically contains:

* `event` the event payload.
* `inputs` any inputs to the node (in the case of resuming a suspended workflow).
* `metadata` event metadata, including the user and  HTTP headers.

HTTP header names are lowercase and only include those that have `x-` as their prefix.

Meta-data will contain the `claimSet/sub` which should always to be used to ensure you only accept events from the correct user. 

Examples:

```
metadata.claimSet.sub == "github" && metadata[`x-github-event`] == "pull_request" && event.repository == "http://gihub.com/argoproj/argo"
```

TODO - we should include several examples

Because the endpoint accepts any JSON data, it is the user's responsibility to write a suitable expression to correctly filter the events they are interested in. Therefore, DO NOT assume the existence of any fields, and guard against them using a nil check:

[Learn more about expression syntax](https://github.com/antonmedv/expr).

```
event.action != nil && event.action.name == "create" 
```
  
## Submitting From A Workflow Template

A workflow template will be submitted (i.e. workflow created from it) and that can be created using parameters from the event itself:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: event-consumer
spec:
  event:
    expression: event.message != "" && metadata["x-argo-e2e"] == ["true"]
  entrypoint: main
  arguments:
    parameters:
      - name: message
        valueFrom:
          expression: event.message
  templates:
    - name: main
      steps:
      - - name: a
          template: argosay
          arguments:
            parameters:
            - name: message
              value: "{{workflow.parameters.message}}"

    - name: argosay
      inputs:
        parameters:
          - name: message
      container:
         image: argoproj/argosay:v2
         args: [echo, "{{inputs.parameters.message}}"]
```

Event:

```bash
curl $ARGO_SERVER/api/v1/events/argo \
    -H "Authorization: $ARGO_TOKEN" \
    -H "X-Argo-E2E: true" \
    -d '{"message": "hello events"}'
```

The resulting workflow prints "hello events".

Submitting is stateless, so you can only have one expression per workflow template. You cannot wait for series of events before submitting.

!!! Warning
    If the expression is malformed, this is only logged. It is not visible in logs or the UI. Use `argo template create` rather than `kubectl apply` to catch your mistakes.

## Resume A Suspended Workflow Node

It is possible to create a workflow that waits for one or more events to occur continuing.

!!! NOTE
    A **suspended workflow** is one just a workflow that has a one or more suspended nodes. Counterintuitively, it can have other running nodes. 

For a workflow to receive an event, it must have at least one suspend node with an event expression that matches that event:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: event-consumer-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
      - - name: a
          template: consume-event
          arguments:
            parameters:
            - name: type
              value: test

    - name: consume-event
      inputs:
        parameters:
          - name: type
      suspend:
        event:
          expression: event.type == inputs.parameters[0].value
      outputs:
        parameters:
          - name: eventType
            valueFrom: 
              expression: event.type
```

```bash
curl $ARGO_SERVER/api/v1/events/argo \
    -H "Authorization: $ARGO_TOKEN" \
    -d '{"type": "test"}'
```

The suspended node will be resumed and the output parameter `eventType` will be set from the node.

If the expression is malformed or evaluates to an error:

* The expression was malformed, or does not evaluate to a boolean: node (and therefore workflow) will be marked as "failed" with the reason.
* The expression could not be evaluated (e.g. by using a field which was not in the event): a warning condition will be added to the workflow
* The output parameters could not be evaluated for any reason: node will be marked as "failed"

## Manual Intervention and Automatic Resumption After Timeout

It should be noted, with suspend nodes you can also:

* Manually resume the node from the user interface or via CLI.
* Automatically resume the node after a duration has passed.

```yaml
suspend:
  duration: 1h
```

## Gate A Cron Workflow

You can use events to gate a cron workflow. For example: schedule the workflow at 1am, and then wait up to 1h for an event.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: event-consumer
spec:
  schedule: "0 1 * * *"
  workflowSpec:
    entrypoint: main
    templates:
      - name: main
        suspend:
          duration: 1h
          event:
            expression: event.type == "test
```

## High-Availability

!!! WARNING
    You MUST run a minimum of two Argo Server replicas if you do not want to lose events. 

If you are processing large numbers of events, you may need to scale up the Argo Server to handle them. 

By default, a single Argo Server can be processing 64 events before the endpoint will start returning 503 errors.

* Vertically you can: 
  * Increase the size of the event pipeline `--event-pipeline-size 16` (good for temporary event bursts).
  * Increase the number of workers `--event-worker-count 4` (good for sustained numbers of events).
* Horizontally you can run more Argo Servers (good for sustained numbers of events AND high-availability).

## Further Reading

If you're sending events from a new system, we recommend Cloud Events:

* [CloudEvents Specification](https://github.com/cloudevents/spec)
* [CloudEvents HTTP Webhook](https://github.com/cloudevents/spec/blob/v1.0/http-webhook.md)
* [Stripe Webhooks](https://stripe.com/docs/webhooks)
* [Github Webhooks](https://developer.github.com/webhooks/)
