# New features in v4.1.0 (2026-07-22)

This is a concise list of new features.

## General

- Add Optional Argo Workflow–Level Configuration for Executor Plugins by [ntny](https://github.com/ntny) ([#15234](https://github.com/argoproj/argo-workflows/issues/15234))
  This PR allows configuring the Argo Workflow Executor Plugin for a specific Argo Workflow directly within the Workflow spec.
  Enable this with the `ARGO_WORKFLOW_LEVEL_EXECUTOR_PLUGINS=true` controller environment variable.
  Workflow-level executor plugin settings take precedence over globally configured executor plugins.
  See `docs/executor_plugins.md` for configuration details and examples.

- improve S3 upload speed with customization of S3 upload threads / partsize by [Antoine Tran](https://github.com/antoinetran) ([#15636](https://github.com/argoproj/argo-workflows/issues/15636))
  By default, the S3 upload code is configured with 4 threads and no fixed part size. The part size is dynamically set by MinIO depending on the max number of chunks and file size, but generally it is 16MiB (for file size <= 156GiB).
  The feature will allow setting the number of threads and part size with env var ARTIFACT_S3_UPLOAD_THREADS and ARTIFACT_S3_UPLOAD_PART_SIZE_MIB.

- add a streaming save path to artifact drivers, including gRPC streaming for plugins by [panicboat](https://github.com/panicboat) ([#12656](https://github.com/argoproj/argo-workflows/issues/12656))
  Artifact drivers can now save an output artifact from an `io.Reader` via a new `SaveStream` method, in addition to saving from a local file path.
  Azure and HTTP/Artifactory stream the reader directly to the destination.
  S3, GCS, OSS, and HDFS buffer the reader to a temp file and reuse their existing save path, so bucket creation, key handling, and retries are unchanged.
  Streaming an HTTP artifact directly cannot follow a 307/308 redirect (e.g. webHDFS), because a one-shot reader cannot be replayed for the redirected request. Set `saveStreamViaFile: true` on the HTTP artifact to buffer the stream to a temp file first, which restores redirect support at the cost of the extra buffering.
  Artifact plugins gain an optional client-streaming `SaveStream` gRPC method plus a `GetCapabilities` method.
  A plugin that advertises support receives the artifact content chunk by chunk with no intermediate temp file.
  A plugin that does not implement these methods keeps working unchanged: Argo buffers the content to a temp file and calls the existing `Save`, so this is backward compatible.

- Support for AWS RDS PostgreSQL IAM authentication by [isubasinghe](https://github.com/isubasinghe) ([#15834](https://github.com/argoproj/argo-workflows/issues/15834))
  Add support for authenticating to AWS RDS PostgreSQL using IAM authentication tokens.
  This allows Argo Workflows to use IAM roles to connect to its persistence database, removing the need for long-lived database passwords.
    - Uses the default AWS credential chain for seamless authentication in AWS environments.
    - Supports optional region override (auto-detected if omitted).
    - Integrates with both persistence and synchronization database configurations.

- Support for Azure PostgreSQL/Entra ID authentication by [isubasinghe](https://github.com/isubasinghe) ([#15530](https://github.com/argoproj/argo-workflows/issues/15530))
  Add support for authenticating to Azure Database for PostgreSQL using Azure AD (Entra ID) tokens.
  This allows Argo Workflows to use Azure Workload Identity or Managed Identity to connect to its persistence database, removing the need for long-lived database passwords.
    - Uses `DefaultAzureCredential` for seamless authentication in Azure environments.
    - Supports configurable token scopes.
    - Integrates with both persistence and synchronization database configurations.

- Allow for configuring the allow list when using workflow template refs. by [Isitha Subasinghe](https://github.com/isubasinghe) ([#16345](https://github.com/argoproj/argo-workflows/issues/16345))
  When the controller runs with `templateReferencing: Strict` or `Secure`, workflows using a `workflowTemplateRef` may only set an allow-listed set of `WorkflowSpec` fields (`arguments`, `entrypoint`, `suspend`, and other benign knobs); every other field is blocked so a submitter cannot override the template's security settings.
  The `WORKFLOW_USER_OVERRIDE_ALLOWLIST` environment variable lets operators add fields to this allow-list.
  Set it on the workflow-controller to a comma-separated list of `WorkflowSpec` field names, using the YAML/JSON names as written in a workflow, for example `WORKFLOW_USER_OVERRIDE_ALLOWLIST=podSpecPatch,volumes`.
  Use this when your environment has decided a normally-blocked field is safe for submitters to override.
  An unknown field name fails the controller at startup rather than being silently ignored, surfacing typos.
  The nested `artifactGC.podSpecPatch`, `artifactGC.serviceAccountName`, and `artifactGC.podMetadata` fields remain blocked and are not relaxed by this variable.

- Configurable node status compression algorithm (gzip, zstd, or brotli) and level by [Isitha Subasinghe](https://github.com/isubasinghe) ([#16262](https://github.com/argoproj/argo-workflows/issues/16262))
  Large workflow node statuses can now be compressed with `zstd` or `brotli` instead of gzip via the `WORKFLOW_COMPRESSION_ALGORITHM` environment variable, with `WORKFLOW_COMPRESSION_LEVEL` tuning the level.
  Decompression auto-detects the algorithm.
  See [node status compression](offloading-large-workflows.md#node-status-compression).

- Disable agent pod creation for plugins by [Gaurang Mishra](https://github.com/gaurang9991) ([#7891](https://github.com/argoproj/argo-workflows/issues/7891))
  Allow users to disable agent pod creation for plugins. Workflow Controller watches the task sets updated by external controllers or agents. Users should be careful when using this: when enabled, it stops creating default agent pods for HTTP templates.

- Reconnect and retry queries by [Isitha Subasinghe](https://github.com/isubasinghe) ([#15011](https://github.com/argoproj/argo-workflows/issues/15011))
  Queries against the database are now retried where a network connection issue was the cause of failure, this
  is done through reconnecting first.

- Hot-reload namespaceParallelism from the controller ConfigMap by [shuangkun](https://github.com/shuangkun) ([#16490](https://github.com/argoproj/argo-workflows/issues/16490))
  Changes to `namespaceParallelism` in the workflow controller ConfigMap now take effect on reload without restarting the controller, matching the existing `parallelism` behavior.
  Namespaces without an explicit parallelism label override track the live default.

- Opt-in pod layout that removes the `argoexec init` container by [Alan Clucas](https://github.com/Joibel) ([#16154](https://github.com/argoproj/argo-workflows/issues/16154))
  New controller-wide `initlessPod` mode (workflow controller ConfigMap) that eliminates the `argoexec init` container. Beta: off by default and may change in incompatible ways in future minor releases before being promoted to stable.
  The `argoexec` binary is mounted into `main` via a Kubernetes image volume (KEP-4639 — Beta in K8s 1.33 behind a feature gate, GA in 1.36), and a new `supervisor` container handles template write, script staging, input artifact download, readiness signaling, and the post-main responsibilities previously held by `wait`.
  Artifact plugins run as regular sidecars invoked by `supervisor` for both Load and Save instead of as init containers, so pods run with zero init containers.
  Off by default; `wait` and the legacy pod layout remain unchanged.
  Enable by setting `initlessPod.enabled: true` in the workflow controller ConfigMap — every subsequently scheduled workflow pod uses the init-less layout.
  Rollback by setting it back to `false`; in-flight pods keep their original layout.

- Add `!=` and `==` operators for namespace field selector by [Miltiadis Alexis](https://github.com/miltalex) ([#13468](https://github.com/argoproj/argo-workflows/issues/13468))
  You can now use the `!=` and `==` operators when filtering workflows by namespace field.
  This provides more flexible query capabilities, allowing you to easily exclude specific namespaces or match exact namespace values in your workflow queries.
  For example, you can filter with `namespace!=kube-system` to exclude system namespaces or `namespace==production` to target only production environments.

- Add pendingTimeout field to templates for setting maximum time in pending status. by [Dennis Lawler](https://github.com/drawlerr) ([#10341](https://github.com/argoproj/argo-workflows/issues/10341))
  Adds a new `pendingTimeout` field to workflow templates that allows setting a maximum duration a pod can spend in Pending status.
  This is useful when pods may be stuck pending due to resource constraints, scheduling issues, or node availability.
  Unlike the existing `timeout` field which covers the entire node lifecycle, `pendingTimeout` specifically targets the pending phase.
  Enforcement is performed by the controller based on its most recently observed pod state, so it is approximate: a pod that
  starts running at almost exactly the moment the pending deadline expires may still be failed and deleted. When the timeout
  fires, the node is marked Failed and the pending pod is deleted to free the resources it was waiting on.

- Pod-level resource requests and limits for workflow pods by [Isitha Subasinghe](https://github.com/isubasinghe) ([#16399](https://github.com/argoproj/argo-workflows/issues/16399))
  Workflow pods can now set [pod-level resource requests and limits](https://kubernetes.io/docs/tasks/configure-pod-container/assign-pod-level-resources/) via the new `podResources` field, available at the workflow spec level and the template level.
  Template-level `podResources` overrides the workflow-level value.
  This lets you set a single resource budget shared by all containers in a pod (main, init, wait and sidecars) instead of sizing each container individually.
  Requires the `PodLevelResources` feature gate to be enabled on the cluster (beta and on by default since Kubernetes v1.34).
  If the feature gate is disabled, the API server strips the field and the controller emits a `PodLevelResourcesDropped` warning event on the workflow.

- S3 virtual-hosted-style bucket addressing by [Himesh Panchal](https://github.com/himeshp) ([#10851](https://github.com/argoproj/argo-workflows/issues/10851))
  S3 artifact storage now supports configuring the bucket addressing style via the `addressingStyle` field.
  Valid values are `""` (auto-detect, default), `"path"` (force path-style), and `"virtual-hosted"` (force virtual-hosted-style).
  This fixes broken log streaming and artifact browsing for S3-compatible providers that only support virtual-hosted-style addressing.

- Workflow Tracing by [Alan Clucas](https://github.com/Joibel) ([#12077](https://github.com/argoproj/argo-workflows/issues/12077))
  Argo Workflows can now emit OpenTelemetry traces, letting you see exactly what's happening inside a workflow run -- from controller reconciliation down to individual artifact uploads and log saves. Traces follow execution across the controller and executor processes, so you get a single span tree covering DAG node scheduling, pod creation, synchronization locks, script capture, and everything in between. If your workloads also emit OTel traces, they'll show up nested in the right place. Configure the tracing section in your workflow-controller-configmap with a collector URL and point your Jaeger or Tempo instance at it.

- Add WorkflowTemplate name as label when using workflowTemplateRef by [Eduardo Rodrigues](https://github.com/eduardodbr) ([#12670](https://github.com/argoproj/argo-workflows/issues/12670))
  When a `Workflow` or a `CronWorkflow` is submitted from a `WorkflowTemplate` or `ClusterWorkflowTemplate` ( i.e. using the `workflowTemplateRef`) it stores the `WorkflowTemplate` name as a label.

## UI

- Allow setting the namespace when submitting a `ClusterWorkflowTemplate` by [Mason Malone](https://github.com/MasonM) ([#10398](https://github.com/argoproj/argo-workflows/issues/10398))
  You can now specify the namespace when submitting a workflow from a `ClusterWorkflowTemplate` using an input field on the "Submit Workflow" panel.

- Upload input artifacts when submitting workflows from the UI by [panicboat](https://github.com/panicboat) ([#12656](https://github.com/argoproj/argo-workflows/issues/12656))
  When a WorkflowTemplate defines input artifacts in `spec.arguments.artifacts`, users can now upload files directly from the UI when submitting the workflow.
  Previously, users had to manually upload files to the artifact repository, know the exact key path, and hard-code the key in the WorkflowTemplate.
  Now, users can simply select a file in the submit dialog.
  The system will upload the file to the artifact repository via the Argo Server, automatically override the artifact key with the uploaded file's location, and submit the workflow with the correct artifact configuration.
  This feature works with all supported artifact repositories (S3, GCS, Azure Blob Storage, OSS, HDFS).
  Uploaded files are written under the `uploads/{namespace}/{uuid}/{filename}` key in the artifact
  repository before the workflow is submitted. The maximum accepted upload size is controlled by the
  `ARGO_SERVER_MAX_ARTIFACT_UPLOAD_BYTES` environment variable on the Argo Server (default `1073741824`,
  i.e., 1 GiB); requests over this size receive `413 Request Entity Too Large`.
  Abandoned uploads (never submitted) rely on operator-configured bucket lifecycle under
  `uploads/{namespace}/`; see [Configuring Your Artifact Repository](configure-artifact-repository.md#abandoned-upload-cleanup).

- Add markdown rendering support to Tooltip component. by [panicboat](https://github.com/panicboat) ([#13936](https://github.com/argoproj/argo-workflows/issues/13936))
  Tooltip now renders markdown content, supporting links, formatting, and line breaks.

- Autocomplete the Namespace filter from the namespaces of resources currently visible on the page by [Morgan Allen](https://github.com/callmemorgan) ([#7405](https://github.com/argoproj/argo-workflows/issues/7405))
  The Namespace filter on the workflows, workflow templates, cron workflows, sensors, event sources, event bindings, and event flow pages now autocompletes from the namespaces of resources already loaded for that page, in addition to the existing localStorage history.
  When the user is viewing resources across multiple namespaces (cluster-wide mode), the namespace filter dropdown now lists the namespaces present on the current page and narrows as you type. No new server endpoint or RBAC is involved — suggestions are derived from data the user already has permission to see.
  The managed-namespace short-circuit (where the filter renders as plain text) is unchanged.

- Show full tag value on hover in TagsInput by [nakatani-yo](https://github.com/nakatani-yo) ([#16096](https://github.com/argoproj/argo-workflows/issues/16096))
  Hovering over truncated tags now shows the full value.

## CLI

- Extend client TLS certificate support in server mode by [Miltiadis Alexis](https://github.com/miltalex) ([#13437](https://github.com/argoproj/argo-workflows/issues/13437))
  Use `--client-certificate` and `--client-key` when an Argo Server or its proxy requires mutual TLS authentication.
  Both flags must be provided together.
  Use `--certificate-authority` to trust the certificate authority that signed the server certificate.
  The certificates are used by the gRPC and HTTP/1 clients, including artifact downloads with `argo cp`.
  For example, run `argo --argo-server argo.example.com:443 --secure --certificate-authority ca.crt --client-certificate client.crt --client-key client.key list`.
  In server mode, client certificates and certificate authorities embedded in a kubeconfig context are not used automatically.
  Pass the three flags explicitly when connecting through Argo Server.
  The flags do not have `ARGO_*` environment variable equivalents.

- Allow archive cli commands to use workflow name instead of uid. by [Isitha Subasinghe](https://github.com/isubasinghe) ([#15199](https://github.com/argoproj/argo-workflows/issues/15199))
  This change allows for `archive` related cli commands to use the workflow name
  instead of relying upon the uid. This is explicitly a user-experience-related improvement.
  Note that if your name itself is a uid, you will have to manually force to fetch via uid or name, see the documentation for more detail.

- Add HTTP proxy support to Argo CLI by [Shimako55](https://github.com/shimako55) ([#10794](https://github.com/argoproj/argo-workflows/issues/10794))
  Add `--proxy-url` flag to Argo CLI commands to support HTTP proxy connections.
  This allows users to connect to Argo Server or Kubernetes API through a corporate proxy or network gateway.
  Works with both Argo Server mode and Kubernetes API mode.
  If `--proxy-url` is not specified, the CLI will respect the standard `HTTP_PROXY` and `HTTPS_PROXY` environment variables.

## Telemetry

- Metrics to observe mutex and semaphore locks, so users can detect unreleased locks that are blocking workflows by [Jason Meridth](https://github.com/jmeridth) [Alan Clucas](https://github.com/Joibel) ([#14888](https://github.com/argoproj/argo-workflows/issues/14888))
  The controller now emits telemetry for synchronization locks (mutexes and semaphores), letting you detect unreleased locks that are causing workflow blocks or timeouts.
  Three metrics are exposed, each labelled by `type` (`mutex` or `semaphore`), `storage` (`configmap` or `database`), `lock_name` and `namespace`:
    - `locks_taken_total` — a counter of how many locks have been acquired, for throughput and churn (`rate()`).
    - `locks_held` — a gauge of how many holders currently hold each lock right now.
    - `locks_pending` — a gauge of how many workflows are currently waiting to acquire each lock.
  For database-backed locks (which are shared across controllers and clusters) each controller reports only its own contribution, using a single per-controller aggregate query per scrape.

- Add metrics for the rate limiter by [Alan Clucas](https://github.com/Joibel) ([#15245](https://github.com/argoproj/argo-workflows/issues/15245))
  Add two rate limiter metrics to help us understand the effects:
    - the k8s API client rate limiter (enabled by default and set quite low, configurable via --qps)
    - and the resource rate limiter configured in the configmap and disabled by default.
  These produce histogram metrics

## Build and Development

- PR readiness helper bot guides contributors through fixing CI failures by [Alan Clucas](https://github.com/Joibel) ([#16231](https://github.com/argoproj/argo-workflows/issues/16231))
  A bot now helps contributors get their PRs ready for review.
  When CI completes on a PR it maintains a single comment listing the contributor-fixable problems — lint, codegen, UI, build, docs, PR title format, missing feature files, DCO sign-off and an unfilled PR description — each with the command to fix it.
  PRs with blocking problems are moved to draft; mark the PR ready for review again once they are fixed.
  The comment updates as checks change and shows all-clear when everything contributor-fixable is resolved.
  Unit and E2E test results are not covered by the bot.
  Maintainers can tune the covered checks and guidance in `.github/pr-readiness/checks.config.json`.
