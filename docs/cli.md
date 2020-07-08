# CLI

The CLI allows to (amongst other things) submit, watch, and list workflows, e.g.: 

```sh
argo submit my-wf.yaml
argo list
```   

## Reference

You can find [detailed reference here](cli/argo.md)

## Help

Most help topics are provided by built-in help:

```
argo --help
```

## Argo Server

You'll need to configure your commands to use the Argo Server if you have [offloaded node status](offloading-large-workflows.md) or are trying to access your [workflow archive](workflow-archive.md). 

To do so, set the `ARGO_SERVER` environment variable, e.g.:

```
export ARGO_SERVER=localhost:2746
```

See [TLS](tls.md).