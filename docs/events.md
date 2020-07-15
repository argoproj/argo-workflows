# Events

> v2.11 and after

![alpha](assets/alpha.svg)

## Overview

To support external webhooks, we have this endpoint `/api/v1/events/{namespace}`. Events can be sent to that can be any JSON data.

These events can:

* [Submit from a workflow template](#submitting-from-a-workflow-template) (or cluster workflow template).

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
  
## Submitting From A Workflow Template

A workflow template will be submitted (i.e. workflow created from it) and that can be created using parameters from the event itself:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: event-consumer
spec:
  event:
    expression: metadata.claimSet.sub == "admin" && event.message != "" && metadata["x-argo-e2e"] == ["true"]
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
metadata.claimSet.sub == "system:serviceaccount:argo:github" && metadata[`x-github-event`] == "pull_request" && event.repository == "http://gihub.com/argoproj/argo"
```

Because the endpoint accepts any JSON data, it is the user's responsibility to write a suitable expression to correctly filter the events they are interested in. Therefore, DO NOT assume the existence of any fields, and guard against them using a nil check:

[Learn more about expression syntax](https://github.com/antonmedv/expr).

## High-Availability

!!! WARNING
    You MUST run a minimum of two Argo Server replicas if you do not want to lose events. 

If you are processing large numbers of events, you may need to scale up the Argo Server to handle them. 

By default, a single Argo Server can be processing 64 events before the endpoint will start returning 503 errors.

* Vertically you can: 
  * Increase the size of the event pipeline `--event-pipeline-size 16` (good for temporary event bursts).
  * Increase the number of workers `--event-worker-count 4` (good for sustained numbers of events).
* Horizontally you can run more Argo Servers (good for sustained numbers of events AND high-availability).
