Description: Add the `{{node.taskName}}` variable and use it in the default artifact key
Authors: [Alan Clucas](https://github.com/Joibel)
Component: General
Issues: 16530

`{{node.taskName}}` is available in container, script, resource, data and container set templates.
For a template that runs in its own pod it has the same value as `{{pod.name}}`: the name of the node's pod, which differs per retry attempt.

The default artifact repository key format is now `{{workflow.name}}/{{node.taskName}}`.
It resolves to the same key as the previous default of `{{workflow.name}}/{{pod.name}}`, so existing artifacts and logs keep their keys.
You can use `{{node.taskName}}` in place of `{{pod.name}}` in your own key formats, for example `keyFormat: "my-artifacts/{{workflow.name}}/{{node.taskName}}"`.

`{{node.taskName}}` names the attempt, while `{{pod.name}}` names its pod.
For a node with its own pod they have the same value, so existing templates and key formats don't need to change.
A pooled template, which runs on hot workers, has no pod of its own and can't use `{{pod.name}}`, so use `{{node.taskName}}` in templates you may pool.
