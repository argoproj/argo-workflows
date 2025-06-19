# New Features

This is a concise list of new features.

## Retry Improvements

* [#13738](https://github.com/argoproj/argo-workflows/pull/13738) : Support retry strategy on daemon containers
* [#13782](https://github.com/argoproj/argo-workflows/pull/13782) : Support cap on retryStrategy backoff
* [#14450](https://github.com/argoproj/argo-workflows/pull/14450) : Allow last retry variables in expressions

## Enhanced Caching

* [#14304](https://github.com/argoproj/argo-workflows/pull/14304) : More granular caching options for the argo kubernetes informer
* [#14205](https://github.com/argoproj/argo-workflows/pull/14205) : Cache semaphore limit lookup

## Security Improvements

* [#14477](https://github.com/argoproj/argo-workflows/pull/14477) : Non-root argoexec image

## UI Enhancements

* [#13962](https://github.com/argoproj/argo-workflows/pull/13962) : Filter workflows by "Finished before" and "Created since" via API
* [#13935](https://github.com/argoproj/argo-workflows/pull/13935) : Allow markdown titles and descriptions in KeyValueEditor
* [#12644](https://github.com/argoproj/argo-workflows/pull/12644) : Allow markdown titles and descriptions in WorkflowTemplates & ClusterWorkflowTemplates
* [#13883](https://github.com/argoproj/argo-workflows/pull/13883) : Mark memoized nodes as cached
* [#13922](https://github.com/argoproj/argo-workflows/pull/13922) : Pre-fill parameters for workflow submit form
* [#14077](https://github.com/argoproj/argo-workflows/pull/14077) : Set template display name in YAML
* [#14034](https://github.com/argoproj/argo-workflows/pull/14034) : Visualize workflows before submitting

## Developer Experience

* [#14412](https://github.com/argoproj/argo-workflows/pull/14412) : Add React Testing Library and initial component coverage
* [#13920](https://github.com/argoproj/argo-workflows/pull/13920) : Move contextless log messages to debug level
* [#14151](https://github.com/argoproj/argo-workflows/pull/14151) : Enable cherry-pick bot

## CLI Enhancements

* [#13999](https://github.com/argoproj/argo-workflows/pull/13999) : Support backfill for cron workflows

## General

* [#14188](https://github.com/argoproj/argo-workflows/pull/14188) : Dynamic namespace parallelism
* [#14103](https://github.com/argoproj/argo-workflows/pull/14103) : Add support for databases enforcing strict data integrity through primary keys
* [#14104](https://github.com/argoproj/argo-workflows/pull/14104) : Label actor action when making changes to workflows/templates
* [#13933](https://github.com/argoproj/argo-workflows/pull/13933) : Support archive logs in resource templates
* [#13790](https://github.com/argoproj/argo-workflows/pull/13790) : Include container name in error messages
* [#14309](https://github.com/argoproj/argo-workflows/pull/14309) : Multi-controller locks (semaphores and mutexes)
