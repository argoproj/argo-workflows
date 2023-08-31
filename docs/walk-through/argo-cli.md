# Argo CLI

In case you want to follow along with this walk-through, here's a quick overview of the most useful argo command line interface (CLI) commands.

```bash
argo submit hello-world.yaml    # submit a workflow spec to Kubernetes
argo list                       # list current workflows
argo get hello-world-xxx        # get info about a specific workflow
argo logs hello-world-xxx       # print the logs from a workflow
argo delete hello-world-xxx     # delete workflow
```

You can also run workflow specs directly [using `kubectl`](../kubectl.md), but the Argo CLI provides syntax checking, nicer output, and requires less typing.

See the [CLI Reference](../cli/argo.md) for more details.
