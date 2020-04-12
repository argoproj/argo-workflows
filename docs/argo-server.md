# Argo Server

![GA](assets/ga.svg)

> v2.5 and after

The Argo Server is a server that exposes an API and UI for workflows. You'll need to use this if you want to [offload large workflows](offloading-large-workflows.md) or the [workflow archive](workflow-archive.md).

You can run this in either "hosted" or "local" mode.

It replaces the Argo UI.

## Hosted Mode

Use this mode if:

* You want a drop-in replacement for the Argo UI.
* If you need to prevent users from directly accessing the database.

Hosted mode is provided as part of the standard [manifests](../manifests), [specifically in argo-server-deployment.yaml](../manifests/base/argo-server/argo-server-deployment.yaml) .

## Local Mode

Use this mode if:

* You want something that does not require complex set-up.
* You do not need to run a database.

To run locally:

```
argo server
```

This will start a server on port 2746 which you can view at [http://localhost:2746](http://localhost:2746]).

## Options

### Auth Mode

You can choose which kube config the server uses:

* "server" - in hosted mode, use the kube config of service account, in local mode, use your local kube config.
* "client" - requires client to provide their Kubernetes bearer token and use that.
* "hybrid" - use the client token if provided, fallback to the server token if note.

By default, the server will start with auth mode of "server".

### Managed Namespace

See [managed namespace](managed-namespace.md).

### Base href

If the server is running behind reverse proxy with a subpath different from `/` (for example, 
`/argo`), you can set an alternative subpath with the `--base-href` flag or the `BASE_HREF` 
environment variable.
