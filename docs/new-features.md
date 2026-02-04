# New features in v4.0.0 (2026-02-04)

This is a concise list of new features.

## General

- Name filter parameter for prefix/contains/exact search in `/archived-workflows` by [Armin Friedl](https://github.com/arminfriedl) ([#14069](https://github.com/argoproj/argo-workflows/issues/14069))
  A new `nameFilter` parameter was added to the `GET
  /archived-workflows` endpoint. The filter works analogous to the one
  in `GET /workflows`. It allows to specify how a search for
  `?listOptions.fieldSelector=metadata.name=<search-string>` in these
  endpoints should be interpreted. Possible values are `Prefix`,
  `Contains` and `Exact`. The `metadata.name` field is matched
  accordingly against the value for `<search-string>`.

- Artifact Drivers as plugins by [Alan Clucas](https://github.com/Joibel), [JP Zivalich](https://github.com/JPZ13), [Elliot Gunton](https://github.com/elliotgunton) ([#5862](https://github.com/argoproj/argo-workflows/issues/5862))
  Artifact Drivers can now be added via a plugin mechanism.
  You can write a GRPC server which acts as an artifact driver to upload and download artifacts to a repository, and supply that as a container image.
  Argo workflows can then use that as a driver.

- Support update total parallelism without restart controller. by [Shuangkun Tian](https://github.com/shuangkun) ([#14689](https://github.com/argoproj/argo-workflows/issues/14689))
  When modify the global parallelism in workflow-controller-configmap, the change takes effect directly without restarting the controller.

- Added CRD validation rules by [Alan Clucas](https://github.com/Joibel ([#13503](https://github.com/argoproj/argo-workflows/issues/13503))
  Added some validation rules to the full CRDs which allow some simpler validation to happen as the object is added to the kubernetes cluster.
  This is useful if you're using a mechanism which bypasses the validator such as kubectl apply.
  It will inform you of
  **Note:** Some validations cannot be implemented as CEL rules due to Kubernetes limitations.
  Fields marked with `+kubebuilder:validation:Schemaless` (like `withItems`) or `+kubebuilder:pruning:PreserveUnknownFields` (like `inline`) are not visible to CEL validation expressions.
  **CEL Budget Management:** Kubernetes limits the total cost of CEL validation rules per CRD. To stay within these limits:
      - All `status` blocks have CEL validations automatically stripped during CRD generation
      - Controller-managed CRDs (WorkflowTaskSet, WorkflowTaskResult, WorkflowArtifactGCTask) have all CEL validations removed from both spec and status
      - Server-side validations in `workflow/validate/validate.go` supplement CEL for fields that cannot be validated with CEL (e.g., schemaless fields)
  **Array and String Size Limits:** To manage CEL validation costs, the following maximum sizes are enforced:
      - Templates per workflow: 200
      - DAG tasks per DAG template: 200
      - Parameters: 500
      - Prometheus metrics per template: 100
      - Gauge metric value string: 256 characters
  **Mutual Exclusivity Rules:**
      - only one template type per template
      - only one of sequence count/end
      - only one of manifest/manifestFrom
      - cannot use both depends and dependencies in DAG tasks.
  **DAG Task Constraints:**
      - task names cannot start with digit when using depends/dependencies
      - cannot use continueOn with depends.
  **Timeout on Non-Leaf Templates:**
      - Timeout cannot be set on steps or dag templates (only on leaf templates).
  **Cron Schedule Format:**
      - CronWorkflow schedules must be valid 5-field cron expressions, specialdescriptors (@yearly, @hourly, etc.), or interval format (@every).
  **Metric Validation:**
      - metric and label names validation
      - help and value fields required
      - real-time gauges cannot use resourcesDuration metrics
  **Artifact:**
      - At most one artifact location may be specified
      - Artifact.Mode must be between 0 and 511 (0777 octal) for file permissions.
  **Enum Validations:**
      - PodGC strategy
      - ConcurrencyPolicy
      - RetryPolicy
      - GaugeOperation
      - Resource action
      - MergeStrategy
        all have restricted allowed values.
  **Name Pattern Constraints:**
      - Template/Step/Task names: max 128 chars, pattern `^[a-zA-Z0-9][-a-zA-Z0-9]*$;`
      - Parameter/Artifact names: pattern `^[a-zA-Z0-9_][-a-zA-Z0-9_]*$.`
  **Minimum Array Sizes:**
      - Template.Steps requires at least one step group
      - Parameter.Enum requires at least one value
      - CronWorkflow.Schedules requires at least one schedule
      - DAG.Tasks requires at least one task.
  **Numeric Constraints:**
      - Parallelism minimum 1
      - StartingDeadlineSeconds minimum 0.

- Allow custom CA certificate configuration for SSO OIDC provider connections by [bradfordwagner](https://github.com/bradfordwagner) ([#7198](https://github.com/argoproj/argo-workflows/issues/7198))
  This feature adds support for custom TLS configuration when connecting to OIDC providers for SSO authentication.
  This is particularly useful when your OIDC provider uses self-signed certificates or custom Certificate Authorities (CAs).
      - Use this feature when your OIDC provider uses custom self-signed CA certificates
      - Configure custom CA certificates either inline or by file path
  **Configuration Examples**
  **Inline PEM content**
      sso:
        # Custom PEM encoded CA certificate file contents
        rootCA: |-
          -----BEGIN CERTIFICATE-----
          MIIDXTCCAkWgAwIBAgIJAKoK/heBjcOuMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
          ...
          -----END CERTIFICATE-----
  The system will automatically use certificates configured with SSL_CERT_DIR, and SSL_CERT_FILE for non macOS environments.
  For production environments, always use proper CA certificates instead of skipping TLS verification.

- Deprecate singular in favour of plural items by [Alan Clucas](https://github.com/Joibel) ([#14977](https://github.com/argoproj/argo-workflows/issues/14977))
  Deprecation: remove singluar `mutex`, `semaphore` and `schedule` from the specs, all were replaced by the plural version in 3.6

- Disable write back informer by default by [Eduardo Rodrigues](https://github.com/eduardodbr) ([#12352](https://github.com/argoproj/argo-workflows/issues/12352))
  Update the controller’s default behavior to disable the write-back informer. We’ve seen several cases of unexpected behavior that appear to be caused by the write-back mechanism, and Kubernetes docs recommend avoiding writes to the informer store. Although turning it off may increase the frequency of 409 Conflict errors, it should help reduce unpredictable controller behavior.

- This migrates most of the logging off logrus and onto a custom logger. by [Isitha Subasinghe](https://github.com/isubasinghe) ([#11120](https://github.com/argoproj/argo-workflows/issues/11120))
  Currently it is quite hard to identify log lines with it's corresponding
  workflow. This change propagates a context object down the call hierarchy
  containing an annotated logging object. This allows context aware logging from
  deep within the codebase.

- Support metadata.name= and metadata.name!= in field selectors by [Miltiadis Alexis](https://github.com/miltalex) ([#13468](https://github.com/argoproj/argo-workflows/issues/13468))
  Field selectors for `metadata.name` now support the `==` and `!=` operators, giving you more flexible control over resource filtering.
  Use the `==` operator to match resources with an exact name, or use `!=` to exclude resources by name.
  This brings field selector behavior in line with native Kubernetes functionality and enables more precise resource queries.

- Restart pods that fail before starting by [Alan Clucas](https://github.com/Joibel) ([#12572](https://github.com/argoproj/argo-workflows/issues/12572))
  Automatically restart pods that fail before starting for reasons like node eviction.
  This is safe to do even for non-idempotent workloads.
  You need to configure this in your workflow controller configmap for it to take effect.

- Removal of logrus and more structured logging by [Alan Clucas](https://github.com/Joibel) ([#11120](https://github.com/argoproj/argo-workflows/issues/11120), [#2308](https://github.com/argoproj/argo-workflows/issues/2308))
  Complete context passing so all logs should have correct context and enable remaining logs to be fully structured logs.

## UI

- Add an informational message to the CronWorkflow delete confirmation modal indicating that Workflows created by the CronSchedule will also be deleted. by [minsun yun](https://github.com/miinsun) ([#14679](https://github.com/argoproj/argo-workflows/issues/14679))
  UI/UX only; **no functional logic** is changed
  Verified manually by deleting a CronWorkflow in the Workflows UI and confirming the message renders correctly

- Add label query parameter sync with URL in WorkflowTemplates UI to match Workflows list behavior for consistent filtering. by [puretension](https://github.com/puretension) ([#14807](https://github.com/argoproj/argo-workflows/issues/14807))
  WorkflowTemplates UI now properly handles label query parameters (e.g., ?label=key%3Dvalue)
  Combined URL updates and localStorage persistence in single useEffect
  Enables custom UI links for filtered template views
  Verified that URL updates when changing filters and filters persist on page refresh

- Name Not Equals filter now available in the UI for filtering workflows by [Miltiadis Alexis](https://github.com/miltalex) ([#13468](https://github.com/argoproj/argo-workflows/issues/13468))
  You can now use the "Name Not Equals" filter in the workflow list to exclude workflows by name.
  This complements the existing "Name Exact" filter and provides more flexible filtering options.
  Use this filter when you want to view all workflows except those matching a specific name pattern.

- Support open custom links in new tab automatically. by [Shuangkun Tian](https://github.com/shuangkun) ([#13114](https://github.com/argoproj/argo-workflows/issues/13114))
  Support configuring a custom link to open in a new tab by default.
  If `target == _blank`, open in new tab, if target is null or `_self`, open in this tab. For example:
      - name: Pod Link
        scope: pod
        target: _blank
        url: http://logging-facility?namespace=${metadata.namespace}&podName=${metadata.name}&startedAt=${status.startedAt}&finishedAt=${status.finishedAt}

- Optimize pagination performance when counting workflows in archive. by [Shuangkun Tian](https://github.com/shuangkun) ([#13948](https://github.com/argoproj/argo-workflows/issues/13948))
  When querying archived workflows with pagination, the system now uses more efficient methods to check if there are more items available. Instead of performing expensive full table scans, the new implementation uses LIMIT queries to check if there are items beyond the current offset+limit, significantly improving performance for large datasets.

## CLI

- Add support for creating a configmap semaphore config using CLI by [Darko Janjic](https://github.com/djanjic) ([#14671](https://github.com/argoproj/argo-workflows/issues/14671))
  Allow user to create a configmap semaphore configuration using CLI

- `convert` CLI command to convert to new workflow format by [Alan Clucas](https://github.com/Joibel) ([#14977](https://github.com/argoproj/argo-workflows/issues/14977))
  A new CLI command `convert` which will convert Workflows, CronWorkflows, and (Cluster)WorkflowTemplates to the new format.
  It will remove `schedule` from CronWorkflows, moving that into `schedules`
  It will remove `mutex` and `semaphore` from `synchronization` blocks and move them to the plural version.
  Otherwise this command works much the same as linting.

- Add support for creating a database semaphore config using CLI by [Darko Janjic](https://github.com/djanjic) ([#14783](https://github.com/argoproj/argo-workflows/issues/14783))
  Allow user to create a database semaphore configuration using CLI

## Telemetry

- Add metrics for the rate limiter by [Alan Clucas](https://github.com/Joibel) ([#15245](https://github.com/argoproj/argo-workflows/issues/15245))
  Add two rate limiter metrics to help us understand the effects:
    - the k8s API client rate limiter (enabled by default and set quite low, configurable via --qps)
    - and the resource rate limiter configured in the configmap and disabled by default.
  These produce histogram metrics

## Build and Development

- Document features as they are created by [Alan Clucas](https://github.com/Joibel) ([#14155](https://github.com/argoproj/argo-workflows/issues/14155))
  To assist with creating release documentation and blog postings, all features now require a document in .features/pending explaining what they do for users.

- Support plural "Authors:" field in feature notes by " field in feature notes ([#14155](https://github.com/argoproj/argo-workflows/issues/14155))
  The featuregen tool now uses the plural "Authors:" field instead of "Author:" for new feature notes.
  This better reflects that features can have multiple authors.
  The change maintains full backwards compatibility with existing feature files that use the singular "Author:" field.
