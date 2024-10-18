# New features

This is a concise list of new features.
See [the upgrade notes](upgrading.md#upgrading_to_v3.6) for information on breaking changes and deprecations.

## UI

* [#13519](https://github.com/argoproj/argo-workflows/pull/13519): The full name of the workflow is now visible in the list details of a workflow.
* [#13284](https://github.com/argoproj/argo-workflows/pull/13284): Various time displays can be switched between relative and absolute ISO time.
* [#10553](https://github.com/argoproj/argo-workflows/pull/10553): You can now use markdown in workflow titles and descriptions and it will be displayed in the UI.
* [#12350](https://github.com/argoproj/argo-workflows/pull/12350): The UI will now show the directory used for input artifacts.
* [#12873](https://github.com/argoproj/argo-workflows/pull/12873): You can also now see line numbers in the object view.
* [#13452](https://github.com/argoproj/argo-workflows/pull/13452): WorkflowTemplate and ClusterWorkflowTemplate will show you their execution history like you can see for CronWorkflows.
* [#12024](https://github.com/argoproj/argo-workflows/pull/12024): You will be able to see live logs from pods if retrieval of logs from archived workflows fails and the pod logs are available.
* [#12674](https://github.com/argoproj/argo-workflows/pull/12674): CronWorkflows and WorkflowTemplates now display their title and descriptions in the list view.
* [#12199](https://github.com/argoproj/argo-workflows/pull/12199): You can specify HTTP headers to use to detect IP addresses using the `IP_KEY_FUNC_HEADERS` environment variable. This is used in the rate limiter.
* [#13695](https://github.com/argoproj/argo-workflows/pull/13695): You can now retry a single node from a workflow, even if the workflow succeeded.
* [#13610](https://github.com/argoproj/argo-workflows/pull/13610): You can now filter with prefixes and patterns in the workflow list.

## Metrics

* [#13265](https://github.com/argoproj/argo-workflows/pull/13265): The workflow controller can now emit metrics over OpenTelemetry GRPC protocol
    * [#13267](https://github.com/argoproj/argo-workflows/pull/13267): with selectable temporality
    * [#13268](https://github.com/argoproj/argo-workflows/pull/13268): configuration of what is emitted
* Many of the metrics have been updated which will require you [to change how you use them](upgrading.md#metrics_changes) and there are some new ones:
    * [#13269](https://github.com/argoproj/argo-workflows/pull/13269): Version information in the controller
    * [#13270](https://github.com/argoproj/argo-workflows/pull/13270): Is this controller the leader
    * [#13271](https://github.com/argoproj/argo-workflows/pull/13271): Kubernetes API calls duration
    * [#13272](https://github.com/argoproj/argo-workflows/pull/13272): Pod phase monitoring
    * [#13274](https://github.com/argoproj/argo-workflows/pull/13274): CronWorkflows counters
    * [#13497](https://github.com/argoproj/argo-workflows/pull/13497): CronWorkflows policy counters
    * [#13275](https://github.com/argoproj/argo-workflows/pull/13275): Workflow Template counters
    * [#13735](https://github.com/argoproj/argo-workflows/pull/13735): Counters to check if you're using deprecated features
* [#11927](https://github.com/argoproj/argo-workflows/pull/11927): There is a new `retries` variable available in metrics describing the number of retries.
* [#11857](https://github.com/argoproj/argo-workflows/pull/11857): Pod missing metrics will be emitted before pods are created

## Controller

* [#13358](https://github.com/argoproj/argo-workflows/pull/13358): You can use multiple mutexes and semaphores in the same workflow or template, and use both type of lock at the same time
* [#13419](https://github.com/argoproj/argo-workflows/pull/13419): The controller uses a queue when archiving workflows to improve memory management when archiving a large number of workflows at once
* [#12441](https://github.com/argoproj/argo-workflows/pull/12441): Plugins can now be stopped, so that a stopped workflow will shutdown its plugin nodes
* The OSS artifact driver:
    * [#12188](https://github.com/argoproj/argo-workflows/pull/12188): Can now work with directories,
    * [#12907](https://github.com/argoproj/argo-workflows/pull/12907): Supports deletion,
    * [#12908](https://github.com/argoproj/argo-workflows/pull/12908): Supports streaming.
* [#12419](https://github.com/argoproj/argo-workflows/pull/12419): Pod deletion will now happen in parallel to speed it up.
* [#13360](https://github.com/argoproj/argo-workflows/pull/13360): You can use Shared Access Signatures to access artifacts stored in Azure.
* [#12413](https://github.com/argoproj/argo-workflows/pull/12413): Workflow pods now have a kubernetes finalizer to try to prevent them being deleted prematurely
* [#12325](https://github.com/argoproj/argo-workflows/pull/12325): Large environment variables will be offloaded to Config Maps
* [#12328](https://github.com/argoproj/argo-workflows/pull/12328): Large and flat workflows where there are many steps that need resolving at the same time could time out during template referencing. This is now much faster.
* [#12568](https://github.com/argoproj/argo-workflows/pull/12568): Kubernetes scheduling constraints such as node selectors and tolerations will now be honored where they are specified in a WorkflowTemplate. These will be applied to the task and step pods.
* [#12984](https://github.com/argoproj/argo-workflows/pull/12984): The pods created by workflows will have a `seccompProfile` of `RuntimeDefault` by default.
* [#12842](https://github.com/argoproj/argo-workflows/pull/12842): You can now template the `name` and `template` in a `templateRef`. This allows for fully data driven workflow DAGs.
* [#13194](https://github.com/argoproj/argo-workflows/pull/13194): The expr library has been upgraded providing some new functions in expressions.
* [#13746](https://github.com/argoproj/argo-workflows/pull/13746): Configuration option to avoid sending kubernetes Events for workflows.
* [#13742](https://github.com/argoproj/argo-workflows/pull/13742): `ARGO_TEMPLATE` environment variable can be configured not to contain input parameters to reduce storage usage.
* [#13745](https://github.com/argoproj/argo-workflows/pull/13745): Added an option to skip workflow duration estimation because it can be expensive.

## Cron Workflows

* [#12616](https://github.com/argoproj/argo-workflows/pull/12616): You can now specify multiple cron schedules on a single CronWorkflow.
* [#12305](https://github.com/argoproj/argo-workflows/pull/12305): You can also use a stop strategy on Cron Workflows to stop them running any more workflows after a set of conditions occur such as too many errors.
* [#13474](https://github.com/argoproj/argo-workflows/pull/13474): Cron Workflows also now have a when expression to further tune which occurrences of the workflow will run and which may be skipped

## CLI

* [#12803](https://github.com/argoproj/argo-workflows/pull/12803): You can now update Cron Workflows, Workflow Templates and Cluster Workflow Templates with the `update` command via the CLI
* [#13364](https://github.com/argoproj/argo-workflows/pull/13364): You can selectively list workflow templates using a `-l` label selector
* [#13128](https://github.com/argoproj/argo-workflows/pull/13128): The CLI will now generate shell completions for the [fish shell](https://fishshell.com/)
* [#12977](https://github.com/argoproj/argo-workflows/pull/12977): We also build and ship the CLI complied for [Risc-V](https://riscv.org/)
* [#12953](https://github.com/argoproj/argo-workflows/pull/12953): The lint command supports a `--no-color` flag
* [#13695](https://github.com/argoproj/argo-workflows/pull/12953): The `--output` flag is now validated

## Build and Development

* [#13000](https://github.com/argoproj/argo-workflows/pull/13000): There is now a `/retest` command for retesting PRs in Github that occasionally fail in a flakey test
* [#12867](https://github.com/argoproj/argo-workflows/pull/12867): You can supply your own HTTP client when using the go API client, allowing for adding a proxy
