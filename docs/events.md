# Events

> v2.11 and after

![alpha](assets/alpha.svg)

## Overview

To support external webhooks, we have this endpoint `/api/v1/events/{namespace}`. Events can be sent to that can be any JSON data.

These events can submit a workflow template (cluster workflow templates are not supported today).

You may also wish to read about [webhooks](webhooks.md).

## Authentication and Security

Clients wanting to send events to the endpoint need an [access token](access-token.md). The token may be namespace-scoped or cluster-scoped.  

```yaml
# Use this role to enable jenkins to submit a workflow template via webhook.
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: jenkins
rules:
- apiGroups:
  - argoproj.io
  resources:
  - workflowtemplates
  verbs:
  - get
- apiGroups:
  - argoproj.io
  resources:
  - workflows
  verbs:
  - create
```

It is only possible to submit workflow templates your access token has access to. 

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

The event endpoint will always return in under 10 seconds because the event will be queued and processed asynchronously. This means you will not be notified synchronously of failure. It will only return a failure (503) if the event processing queue is full.  

!!! Warning "Processing Order"
    Events may not always be processed in the order they are received.   
  
## Submitting From A Workflow Template

A workflow template will be submitted (i.e. workflow created from it) and that can be created using parameters from the event itself. 
The following example will be trigger by an event from "admin" with "message" in the payload. That message will be used as an argument for the created workflow.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: event-consumer
spec:
  event:
    expression: metadata.claimSet.sub == "admin" && payload.message != "" && metadata["x-argo"] == ["true"] && discriminator == "my-discriminator"
    parameters:
      - name: message
        expression: payload.message
  entrypoint: main
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
curl $ARGO_SERVER/api/v1/events/argo/my-discriminator \
    -H "Authorization: $ARGO_TOKEN" \
    -H "X-Argo-E2E: true" \
    -d '{"message": "hello events"}'
```

The resulting workflow prints "hello events".

!!! Warning "Malformed Expressions"
    If the expression is malformed, this is logged. It is not visible in logs or the UI. Use `argo template create` rather than `kubectl apply` to catch your mistakes.

## Event Expression Syntax and the Event Expression Environment

**Event expressions** are expressions that are evaluated over the **event expression environment**.

### Expression Syntax

Because the endpoint accepts any JSON data, it is the user's responsibility to write a suitable expression to correctly filter the events they are interested in. Therefore, DO NOT assume the existence of any fields, and guard against them using a nil check.

[Learn more about expression syntax](https://github.com/antonmedv/expr).

### Expression Environment

The event environment typically contains:

* `payload` the event payload.
* `metadata` event metadata, including the user info and HTTP headers.
* `discriminator` the discriminator from the URL.  

### Payload

This is the JSON payload of the event.

Example:

```
payload.repository.clone_url == "http://gihub.com/argoproj/argo"
```

### MetaData 

Metadata is data about the event, this includes **headers** and the **claim set**:

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

#### ClaimSet

Meta-data will contain the value `claimSet/sub` which should always to be used to ensure you only accept events from the correct client. 

Example:

```
metadata.claimSet.sub == "system:serviceaccount:argo:jenkins"
```

### Discriminator

This is only for edge-cases where neither the claim-set subject, payload, or metadata provide enough information to discriminate. Typically, it should be empty and ignored.

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
