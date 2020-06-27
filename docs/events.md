# Events

> v2.10 and after

![alpha](assets/alpha.svg)

It is possible to create a workflow that waits for one or more event to occur continuing.

This allows you to wait for webhooks or other external events.  

* [Example Workflow](../examples/events.yaml)

If you're sending events from a new system, we recommend Cloud Events:

* [CloudEvents Specification](https://github.com/cloudevents/spec)
* [CloudEvents HTTP Webhook](https://github.com/cloudevents/spec/blob/v1.0/http-webhook.md)