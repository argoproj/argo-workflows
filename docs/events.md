# Events

> v2.10 and after

![alpha](assets/alpha.svg)

It is possible to create a workflow that waits for one or more events to occur continuing.

Events are recieved on the web hook endpoint (`/api/v1/events/{namespace}`) and can be any JSON data:

```shell script
curl -d '{"type": "test"}' https://localhost:2746/api/v1/events/argo -k -H "Authorization: Bearer $ARGO_TOKEN"
```

For a workflow to recieve an event, it must have a suspend node with a event expressison that matches that event:

```yaml
suspend:
  event:
    expression: event.type == "test"
```

A suspended node can match an event in the following ways:

* The expression evaluate errors (due to invalid expression), or does not evaluate to a boolean: node will be marked as "failed".
* The expression evaluates to true: the node will be marked as "successful" and the workflow will resume.






* [Example Workflow](../examples/event-consumer.yaml)

If you're sending events from a new system, we recommend Cloud Events:

* [CloudEvents Specification](https://github.com/cloudevents/spec)
* [CloudEvents HTTP Webhook](https://github.com/cloudevents/spec/blob/v1.0/http-webhook.md)

