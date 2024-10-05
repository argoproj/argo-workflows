# Workflow Notifications

There are a number of use cases where you may wish to notify an external system when a workflow completes:

1. Send an email.
1. Send a Slack (or other instant message).
1. Send a message to Kafka (or other message bus).

You have options:

1. For individual workflows, can add an exit handler to your workflow, such as in [this example](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/exit-handlers.yaml).
1. If you want the same for every workflow, you can add an exit handler to [the default workflow spec](default-workflow-specs.md).
1. Use a service (e.g. [Heptio Labs EventRouter](https://github.com/heptiolabs/eventrouter)) to the [Workflow events](workflow-events.md) we emit.
