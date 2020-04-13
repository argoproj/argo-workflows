# CLI

Most help topics are provided using built-in help

```
argo --help
```

# Argo Server

![GA](assets/ga.svg)

> v2.5 and after

You'll need to configure your commands to use the Argo Server if you have [offloaded node status](offloading-large-workflows.md) or are trying to access your [workflow archive](workflow-archive.md). 

To do so, set the ARGO_SERVER environment variable, e.g.:

```
export ARGO_SERVER=localhost:2746
```
