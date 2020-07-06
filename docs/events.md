# Events

> v2.10 and after

![alpha](assets/alpha.svg)

## Submitting A Workflow Template

A workflow template can be submitted by an event that it matches:

```yaml
spec:
  event:
    expression: event.type == "test"
```

Events are accepted on the web hook endpoint (`/api/v1/events/{namespace}`) and can be any JSON data. It is the user's responsibility to write a suitable expression to correctly filter the events they are interested in. 

It is not possible for a workflow template to be submitted only for a series of events (e.g. only trigger ) 


```shell script
curl -d '{"type": "test"}' https://localhost:2746/api/v1/events/argo -k -H "Authorization: Bearer $ARGO_TOKEN"
```

## Resuming A Workflow

It is possible to create a workflow that waits for one or more events to occur continuing.


For a workflow to receive an event, it must have at least one suspend node with an event expression that matches that event:

```yaml
suspend:
  event:
    expression: event.type == "test"
```

A suspended node can match an event in the following ways:

* The expression evaluate errors (due to invalid expression), or does not evaluate to a boolean: node will be marked as "failed".
* The expression evaluates to true: the node will be marked as "successful" and the workflow will resume.

* [Example Workflow](../examples/event-consumer.yaml)

## Further Reading

If you're sending events from a new system, we recommend Cloud Events:

* [CloudEvents Specification](https://github.com/cloudevents/spec)
* [CloudEvents HTTP Webhook](https://github.com/cloudevents/spec/blob/v1.0/http-webhook.md)

