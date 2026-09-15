Description: Add a `node.type` variable available to all templates, usable in `retryStrategy.expression` to filter retries by node type
Authors: [shuangkun](https://github.com/shuangkun)
Component: General
Issues: 13990

The `node.type` variable exposes the type of the current node (`Pod`, `Steps`, `DAG`, `Suspend`, `HTTP`, `Plugin`) to every template, alongside the existing `node.name`.

Because retry expressions have access to all-template variables, this lets a `retryStrategy.expression` decide whether to retry based on the node type.
For example, only retry pod nodes:

```yaml
retryStrategy:
  expression: node.type == "Pod"
```

Inside a retry expression, `node.type` is the type of the node being retried.
