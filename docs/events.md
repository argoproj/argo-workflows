# Events

## Overview

To support external webhooks, we have this endpoint `/api/v1/events/{namespace}/{discriminator}` on the Argo Server.
Events sent to that can be any JSON data.

These Events can submit *WorkflowTemplates* or *ClusterWorkflowTemplates*.

You may also wish to read about [webhooks](webhooks.md).

It should be noted that despite the similarity in name, this feature has nothing to do with [Argo Events](https://argoproj.github.io/argo-events/).

## Authentication and Security

Clients wanting to send events to the endpoint need an [access token](access-token.md).

It is only possible to submit WorkflowTemplates that your access token has access to: [example role](https://raw.githubusercontent.com/argoproj/argo-workflows/main/manifests/quick-start/base/webhooks/submit-workflow-template-role.yaml).

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

The Event endpoint will always return in under 10 seconds because the Event will be queued and processed asynchronously.
This means you will not be notified synchronously of failure.
It will return a failure (503) if the Event processing queue is full.

!!! Warning "Processing Order"
    Events may not always be processed in the order they are received.

## WorkflowTemplate triggered by the event

Before the binding between an event and a WorkflowTemplate, you must create the WorkflowTemplate that you want to trigger.
The following one takes in input the "message" parameter specified into the API call body, passed through the `WorkflowEventBinding` parameters section, and finally resolved here as the message of the `main` template.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: my-wf-tmple
  namespace: argo
spec:
  templates:
    - name: main
      inputs:
        parameters:
          - name: message
            value: "{{workflow.parameters.message}}"
      container:
        image: busybox
        command: [echo]
        args: ["{{inputs.parameters.message}}"]
  entrypoint: main
```

## Submitting A Workflow From A WorkflowTemplate

A WorkflowTemplate will be submitted (i.e. Workflow created from it) and that can be created using parameters from the event itself.
The following example will be triggered by an event with "message" in the payload.
That message will be used as an argument for the created Workflow.
Note that the name of the meta-data header "x-argo-e2e" is lowercase in the selector to match.
Incoming header names are converted to lowercase.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowEventBinding
metadata:
  name: event-consumer
spec:
  event:
    # metadata header name must be lowercase to match in selector
    selector: payload.message != "" && metadata["x-argo-e2e"] == ["true"] && discriminator == "my-discriminator"
  submit:
    workflowTemplateRef:
      name: my-wf-tmple
    arguments:
      parameters:
      - name: message
        valueFrom:
          event: payload.message
```

Please, notice that `workflowTemplateRef` refers to a WorkflowTemplate with the name `my-wf-tmple`, this WorkflowTemplate has to be created before the triggering of the event.
After that you have to apply the above explained `WorkflowEventBinding` (in this example this is called `event-template.yml`) to realize the binding between WorkflowTemplate and event (you can use `kubectl` to do that):

```bash
kubectl apply -f event-template.yml
```

Finally you can trigger the creation of your first parametrized WorkflowTemplate, by using the following call:

Event:

```bash
curl $ARGO_SERVER/api/v1/events/argo/my-discriminator \
    -H "Authorization: $ARGO_TOKEN" \
    -H "X-Argo-E2E: true" \
    -d '{"message": "hello events"}'
```

!!! Warning "Malformed Expressions"
    If the expression is malformed, this is logged.
    It is not visible in logs or the UI.

### Customizing the Workflow Meta-Data

You can customize the name of the submitted Workflow as well as add Annotations and Labels.
This is done by adding a `metadata` object to the Submit object.

Normally the name of the Workflow created from an Event is simply the name of the WorkflowTemplate with a time-stamp appended.
This can be customized by setting the name in the `metadata` object.

Annotations and Labels are added in the same fashion.

All the values for the Name, Annotations and Labels are treated as expressions (see below for details).
The `metadata` object is the same `metadata` type as on all Kubernetes resources and as such is parsed in the same manner.
It is best to enclose the expression in single quotes to avoid any problems when submitting the WorkflowEventBinding to Kubernetes.

This is an example snippet of how to set the name, annotations and labels.
This is based on the Workflow binding from above, and the first Event.

```yaml
submit:
  metadata:
    annotations:
      anAnnotation: 'event.payload.message'
    name: 'event.payload.message + "-world"'
    labels:
      someLabel: '"literal string"'
```

This will result in the Workflow being named "hello-world" instead of `my-wf-tmple-<timestamp>`.
There will be an extra Label with the key `someLabel` and a value of "literal string".
There will also be an extra Annotation with the key `anAnnotation` and a value of "hello"

Be careful when setting the name.
If the name expression evaluates to that of a currently existing Workflow, the new Workflow will fail to submit.

The name, Annotation and Label expression must evaluate to a string and follow the normal [Kubernetes naming requirements](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/).

## Event Expression Syntax and the Event Expression Environment

**Event expressions**, such as the `.spec.event.selector` or `...valueFrom.event` fields, are [expressions](variables.md#expression) that are evaluated over the **event expression environment**.

### Expression Syntax

Because the endpoint accepts any JSON data, it is the user's responsibility to write a suitable expression to correctly filter the events they are interested in.
Therefore, DO NOT assume the existence of any fields, and guard against them using a nil check.

### Expression Environment

The event environment contains:

* `payload` the Event payload.
* `metadata` event metadata, including HTTP headers.
* `discriminator` the discriminator from the URL.

### Payload

This is the JSON payload of the event.

Example:

```text
payload.repository.clone_url == "http://github.com/argoproj/argo"
```

### Metadata

Metadata is data about the Event, this includes **headers**:

#### Headers

HTTP header names are lowercase and only include those that have `x-` as their prefix.
Their values are lists, not single values.

* Wrong: `metadata["X-Github-Event"] == "push"`
* Wrong: `metadata["x-github-event"] == "push"`
* Wrong: `metadata["X-Github-Event"] == ["push"]`
* Wrong: `metadata["github-event"] == ["push"]`
* Wrong: `metadata["authorization"] == ["push"]`
* Right: `metadata["x-github-event"] == ["push"]`

Example:

```text
metadata["x-argo"] == ["yes"]
```

### Discriminator

This is only for edge-cases where neither the payload, or metadata provide enough information to discriminate.
Typically, it should be empty and ignored.

Example:

```text
discriminator == "my-discriminator"
```

## High-Availability

!!! Warning "Run Minimum 2 Replicas"
    You MUST run a minimum of two Argo Server replicas if you do not want to lose Events.

If you are processing large numbers of events, you may need to scale up the Argo Server to handle them.
By default, a single Argo Server can be processing 64 events before the endpoint will start returning 503 errors.

Vertically you can:

* Increase the size of the event operation queue `--event-operation-queue-size` (good for temporary event bursts).
* Increase the number of workers `--event-worker-count` (good for sustained numbers of events).

Horizontally you can:

* Run more Argo Servers (good for sustained numbers of Events AND high-availability).
