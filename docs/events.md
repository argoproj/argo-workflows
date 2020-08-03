# Events

> v2.11 and after

![alpha](assets/alpha.svg)

## Overview

To support external webhooks, we have this endpoint `/api/v1/events/{namespace}/{descriminator}`. Events can be sent to that can be any JSON data.

These events can submit *workflow templates* or *cluster workflow templates*.

You may also wish to read about [webhooks](webhooks.md).

## Authentication and Security

Clients wanting to send events to the endpoint need an [access token](access-token.md).   

It is only possible to submit workflow templates your access token has access to: [example role](manifests/quick-start/base/webhooks/submit-workflow-template-role.yaml).

Example (note the trailing slash):

```bash
curl https://localhost:2746/api/v1/events/argo/ \
  -H "Authorization: $ARGO_TOKEN" \
  -d '{"message": "hello"}'
```

With a **discriminator**:

```bash
curl https://localhost:2746/api/v1/events/argo/my-discriminator \
  -H "Authorization: $ARGO_TOKEN" \
  -d '{"message": "hello"}'
```

The event endpoint will always return in under 10 seconds because the event will be queued and processed asynchronously. This means you will not be notified synchronously of failure. It will return a failure (503) if the event processing queue is full.  

!!! Warning "Processing Order"
    Events may not always be processed in the order they are received.   
  
## Submitting A Workflow From A Workflow Template

A workflow template will be submitted (i.e. workflow created from it) and that can be created using parameters from the event itself. 
The following example will be trigger by an event with "message" in the payload. That message will be used as an argument for the created workflow.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowEventBinding
metadata:
  name: event-consumer
spec:
  event:
    selector: payload.message != "" && metadata["x-argo"] == ["true"] && discriminator == "my-discriminator"
  submit:
    workflowTemplateRef:
      name: my-wf-tmple
    arguments:
      parameters:
      - name: message
        valueFrom:
          event: payload.message
```

Event:

```bash
curl $ARGO_SERVER/api/v1/events/argo/my-discriminator \
    -H "Authorization: $ARGO_TOKEN" \
    -H "X-Argo-E2E: true" \
    -d '{"message": "hello events"}'
```

!!! Warning "Malformed Expressions"
    If the expression is malformed, this is logged. It is not visible in logs or the UI. 

## Event Expression Syntax and the Event Expression Environment

**Event expressions** are expressions that are evaluated over the **event expression environment**.

### Expression Syntax

Because the endpoint accepts any JSON data, it is the user's responsibility to write a suitable expression to correctly filter the events they are interested in. Therefore, DO NOT assume the existence of any fields, and guard against them using a nil check.

[Learn more about expression syntax](https://github.com/antonmedv/expr).

### Expression Environment

The event environment contains:

* `payload` the event payload.
* `metadata` event metadata, including HTTP headers.
* `discriminator` the discriminator from the URL.  

### Payload

This is the JSON payload of the event.

Example:

```
payload.repository.clone_url == "http://gihub.com/argoproj/argo"
```

### MetaData 

Metadata is data about the event, this includes **headers**:

#### Headers

HTTP header names are lowercase and only include those that have `x-` as their prefix. Their values are lists, not single values.    

* Wrong: `metadata["X-Github-Event"] = "push"`
* Wrong: `metadata["x-github-event"] = "push"`
* Wrong: `metadata["X-Github-Event"] = ["push"]`
* Wrong: `metadata["github-event"] = ["push"]`
* Wrong: `metadata["authorization"] = ["push"]`
* Right: `metadata["x-github-event"] = ["push"]`

Example:

```
metadata["x-argo"] == ["yes"]
```

### Discriminator

This is only for edge-cases where neither the payload, or metadata provide enough information to discriminate. Typically, it should be empty and ignored.

Example:

```
discriminator == "my-discriminator"
```

## High-Availability

!!! Warning "Run Minimum 2 Replicas"
    You MUST run a minimum of two Argo Server replicas if you do not want to lose events. 

If you are processing large numbers of events, you may need to scale up the Argo Server to handle them. By default, a single Argo Server can be processing 64 events before the endpoint will start returning 503 errors.

Vertically you can:
 
* Increase the size of the event operation queue `--event-operation-queue-size` (good for temporary event bursts).
* Increase the number of workers `--event-worker-count` (good for sustained numbers of events).

Horizontally you can:
 
* Run more Argo Servers (good for sustained numbers of events AND high-availability).
