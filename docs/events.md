# Events

## Overview

To support external webhooks, the Argo Server has the endpoint `/api/v1/events/{namespace}/{discriminator}`.
Events sent to that endpoint can be any JSON data.

These Events can submit *WorkflowTemplates* or *ClusterWorkflowTemplates*.

You may also wish to read about [webhooks](webhooks.md).

Despite the similar name, this feature is not related to [Argo Events](https://argoproj.github.io/argo-events/).

## Authentication and Security

Clients wanting to send events to the endpoint need an [access token](access-token.md).

You can only submit WorkflowTemplates that your access token has access to: [example role](https://raw.githubusercontent.com/argoproj/argo-workflows/main/manifests/quick-start/base/webhooks/submit-workflow-template-role.yaml).

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

By default, the Event endpoint processes the Event before it returns, and can report failures to you.
If you enable `--event-async-dispatch`, the endpoint queues the Event and returns without waiting for Argo to process it.
This means that Argo does not notify you of failures synchronously.
With async dispatch, the endpoint returns a failure (503) if the Event processing queue is full.

!!! Warning "Processing Order"
    Argo does not always process Events in the order it receives them.

## WorkflowTemplate triggered by the event

Before you bind an event to a WorkflowTemplate, you must create the WorkflowTemplate that you want to trigger.
The following WorkflowTemplate takes as input the "message" parameter from the API call body.
The `WorkflowEventBinding` parameters section passes the parameter through, and the `main` template uses it as its message.

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

An event submits a WorkflowTemplate (that is, creates a Workflow from it), and can pass parameters from the event to the Workflow.
An event with "message" in the payload triggers the following example.
The created Workflow uses that message as an argument.
Note that the name of the metadata header "x-argo-e2e" is lowercase in the selector to match.
Argo converts incoming header names to lowercase.

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

The `workflowTemplateRef` refers to a WorkflowTemplate named `my-wf-tmple`.
You must create this WorkflowTemplate before you trigger the event.
Then apply the above `WorkflowEventBinding` (called `event-template.yml` in this example) to bind the WorkflowTemplate to the event, for example with `kubectl`:

```bash
kubectl apply -f event-template.yml
```

Finally, you can trigger your parameterized WorkflowTemplate with the following call:

Event:

```bash
curl $ARGO_SERVER/api/v1/events/argo/my-discriminator \
    -H "Authorization: $ARGO_TOKEN" \
    -H "X-Argo-E2E: true" \
    -d '{"message": "hello events"}'
```

!!! Warning "Malformed Expressions"
    If an expression is malformed, the Argo Server logs the error and records a `WorkflowEventBindingError` Kubernetes Event on the WorkflowEventBinding.
    By default, the endpoint also returns the error to you.

### Customizing the Workflow Meta-Data

You can customize the name of the submitted Workflow as well as add Annotations and Labels.
To do this, add a `metadata` object to the Submit object.

By default, the name of the Workflow created from an Event is the name of the WorkflowTemplate with a random suffix appended.
You can customize it by setting the name in the `metadata` object.

You add Annotations and Labels the same way.

Argo treats all the values for the Name, Annotations and Labels as expressions (see below for details).
The `metadata` object is the same `metadata` type as on all Kubernetes resources, and Argo parses it the same way.
Enclose the expression in single quotes to avoid problems when you submit the WorkflowEventBinding to Kubernetes.

The following snippet shows how to set the name, annotations and labels.
It builds on the preceding WorkflowEventBinding and the first Event.

```yaml
submit:
  metadata:
    annotations:
      anAnnotation: 'event.payload.message'
    name: 'event.payload.message + "-world"'
    labels:
      someLabel: '"literal string"'
```

With this metadata, the first Event creates a Workflow named "hello-world" instead of `my-wf-tmple-<random-suffix>`.
The Workflow has an extra Label with the key `someLabel` and a value of "literal string".
It also has an extra Annotation with the key `anAnnotation` and a value of "hello".

Be careful when setting the name.
If the name expression evaluates to the name of an existing Workflow, the new Workflow fails to submit.

The name, Annotation and Label expression must evaluate to a string and follow the normal [Kubernetes naming requirements](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/).

## Event Expression Syntax and the Event Expression Environment

**Event expressions**, such as the `.spec.event.selector` or `...valueFrom.event` fields, are [expressions](variables.md#expression) that Argo evaluates against the **event expression environment**.

### Expression Syntax

Because the endpoint accepts any JSON data, you are responsible for writing an expression that correctly filters the events you are interested in.
Do not assume that any field exists, and guard against missing fields with a nil check.

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

Metadata is data about the Event, including **headers**:

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

This is only for edge-cases where neither the payload nor the metadata provides enough information to discriminate.
Typically, you leave it empty and ignore it.

Example:

```text
discriminator == "my-discriminator"
```

## High-Availability

!!! Warning "Run Minimum 2 Replicas"
    You MUST run a minimum of two Argo Server replicas if you do not want to lose Events.

If you are processing large numbers of events, you may need to scale up the Argo Server to handle them.
If you enable `--event-async-dispatch`, a single Argo Server can queue 16 events by default before the endpoint starts returning 503 errors.

If you enable `--event-async-dispatch`, you can scale vertically:

* Increase the size of the event operation queue `--event-operation-queue-size` (good for temporary event bursts).
* Increase the number of workers `--event-worker-count` (good for sustained numbers of events).

Horizontally you can:

* Run more Argo Servers (good for sustained numbers of Events AND high-availability).
