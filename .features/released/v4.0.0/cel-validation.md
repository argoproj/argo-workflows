Description: Added CRD validation rules
Authors: [Alan Clucas](https://github.com/Joibel
Component: General
Issues: 13503

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
