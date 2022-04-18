# Field Reference

## Workflow

Workflow is the definition of a workflow resource

<details>
<summary>Examples (click to open)</summary>
<br>

- [`archive-location.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-artifacts.yaml)

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`arguments-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`artifact-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`artifact-repository-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-repository-ref.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`workflow-template-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref.yaml)

- [`coinflip-recursive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip.yaml)

- [`colored-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/colored-logs.yaml)

- [`conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-artifacts.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`conditionals-complex.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals-complex.yaml)

- [`conditionals.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals.yaml)

- [`graph-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/graph-workflow.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`parallel-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/parallel-workflow.yaml)

- [`sequence-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/sequence-workflow.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/continue-on-fail.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`daemoned-stateful-set-with-service.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemoned-stateful-set-with-service.yaml)

- [`dag-coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-coinflip.yaml)

- [`dag-conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-artifacts.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`dag-continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-continue-on-fail.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-disable-failFast.yaml)

- [`dag-enhanced-depends.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-enhanced-depends.yaml)

- [`dag-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflow.yaml)

- [`dag-multiroot.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-nested.yaml)

- [`dag-targets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-targets.yaml)

- [`dag-task-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-task-level-timeout.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`default-pdb-support.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/default-pdb-support.yaml)

- [`dns-config.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dns-config.yaml)

- [`exit-code-output-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-code-output-variable.yaml)

- [`exit-handler-dag-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-dag-level.yaml)

- [`exit-handler-slack.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-slack.yaml)

- [`exit-handler-step-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-step-level.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`exit-handlers.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handlers.yaml)

- [`expression-destructure-json.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-destructure-json.yaml)

- [`expression-reusing-verbose-snippets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-reusing-verbose-snippets.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`forever.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/forever.yaml)

- [`fun-with-gifs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/gc-ttl.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)

- [`global-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)

- [`hello-hybrid.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-hybrid.yaml)

- [`hello-windows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-windows.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-world.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`http-success-condition.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-success-condition.yaml)

- [`image-pull-secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`init-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-s3.yaml)

- [`k8s-jobs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-jobs.yaml)

- [`k8s-json-patch-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-json-patch-workflow.yaml)

- [`k8s-orchestration.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-orchestration.yaml)

- [`k8s-owner-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-owner-reference.yaml)

- [`k8s-patch-basic.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-patch-basic.yaml)

- [`k8s-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-patch.yaml)

- [`k8s-resource-log-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-resource-log-selector.yaml)

- [`k8s-set-owner-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-set-owner-reference.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`key-only-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/key-only-artifact.yaml)

- [`label-value-from-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/label-value-from-workflow.yaml)

- [`life-cycle-hooks-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-tmpl-level.yaml)

- [`life-cycle-hooks-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-wf-level.yaml)

- [`loops-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-dag.yaml)

- [`loops-maps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-sequence.yaml)

- [`loops.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`node-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/node-selector.yaml)

- [`output-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-gcs.yaml)

- [`output-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-gc-strategy-with-label-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy-with-label-selector.yaml)

- [`pod-gc-strategy.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy.yaml)

- [`pod-metadata-wf-field.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata-wf-field.yaml)

- [`pod-metadata.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-yaml-patch.yaml)

- [`recursive-for-loop.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/recursive-for-loop.yaml)

- [`resource-delete-with-flags.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/resource-delete-with-flags.yaml)

- [`resource-flags.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/resource-flags.yaml)

- [`resubmit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/resubmit.yaml)

- [`retry-backoff.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-backoff.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`retry-container-to-completion.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container.yaml)

- [`retry-on-error.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-on-error.yaml)

- [`retry-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-script.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-python.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)

- [`selected-executor-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/selected-executor-workflow.yaml)

- [`sidecar-dind.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar.yaml)

- [`status-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/status-reference.yaml)

- [`step-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/step-level-timeout.yaml)

- [`steps-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps-inline-workflow.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)

- [`suspend-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template.yaml)

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`synchronization-mutex-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-wf-level.yaml)

- [`template-defaults.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-defaults.yaml)

- [`template-on-exit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-on-exit.yaml)

- [`timeouts-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-pvc.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)

- [`workflow-of-workflows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-of-workflows.yaml)

- [`dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/steps.yaml)

- [`workflow-archive-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-archive-logs.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`workflow-template-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiVersion`|`string`|APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources|
|`kind`|`string`|Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds|
|`metadata`|[`ObjectMeta`](#objectmeta)|_No description available_|
|`spec`|[`WorkflowSpec`](#workflowspec)|_No description available_|
|`status`|[`WorkflowStatus`](#workflowstatus)|_No description available_|

## CronWorkflow

CronWorkflow is the definition of a scheduled workflow resource

<details>
<summary>Examples (click to open)</summary>
<br>

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`cron-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-workflow.yaml)

- [`dag-inline-cronworkflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-cronworkflow.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiVersion`|`string`|APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources|
|`kind`|`string`|Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds|
|`metadata`|[`ObjectMeta`](#objectmeta)|_No description available_|
|`spec`|[`CronWorkflowSpec`](#cronworkflowspec)|_No description available_|
|`status`|[`CronWorkflowStatus`](#cronworkflowstatus)|_No description available_|

## WorkflowTemplate

WorkflowTemplate is the definition of a workflow template resource

<details>
<summary>Examples (click to open)</summary>
<br>

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`dag-inline-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflowtemplate.yaml)

- [`event-consumer-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-event-binding/event-consumer-workflowtemplate.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiVersion`|`string`|APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#resources|
|`kind`|`string`|Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.io.k8s.community/contributors/devel/sig-architecture/api-conventions.md#types-kinds|
|`metadata`|[`ObjectMeta`](#objectmeta)|_No description available_|
|`spec`|[`WorkflowSpec`](#workflowspec)|_No description available_|

## WorkflowSpec

WorkflowSpec is the specification of a Workflow.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-artifacts.yaml)

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`arguments-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`artifact-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`artifact-repository-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-repository-ref.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`workflow-template-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref.yaml)

- [`coinflip-recursive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip.yaml)

- [`colored-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/colored-logs.yaml)

- [`conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-artifacts.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`conditionals-complex.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals-complex.yaml)

- [`conditionals.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals.yaml)

- [`graph-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/graph-workflow.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`parallel-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/parallel-workflow.yaml)

- [`sequence-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/sequence-workflow.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/continue-on-fail.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`cron-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-coinflip.yaml)

- [`dag-conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-artifacts.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`dag-continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-continue-on-fail.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-disable-failFast.yaml)

- [`dag-enhanced-depends.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-enhanced-depends.yaml)

- [`dag-inline-clusterworkflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-clusterworkflowtemplate.yaml)

- [`dag-inline-cronworkflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-cronworkflow.yaml)

- [`dag-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflow.yaml)

- [`dag-inline-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflowtemplate.yaml)

- [`dag-multiroot.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-nested.yaml)

- [`dag-targets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-targets.yaml)

- [`dag-task-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-task-level-timeout.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`default-pdb-support.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/default-pdb-support.yaml)

- [`dns-config.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dns-config.yaml)

- [`exit-code-output-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-code-output-variable.yaml)

- [`exit-handler-dag-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-dag-level.yaml)

- [`exit-handler-slack.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-slack.yaml)

- [`exit-handler-step-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-step-level.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`exit-handlers.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handlers.yaml)

- [`expression-destructure-json.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-destructure-json.yaml)

- [`expression-reusing-verbose-snippets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-reusing-verbose-snippets.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`forever.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/forever.yaml)

- [`fun-with-gifs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/gc-ttl.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)

- [`global-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)

- [`hello-hybrid.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-hybrid.yaml)

- [`hello-windows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-windows.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-world.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`http-success-condition.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-success-condition.yaml)

- [`image-pull-secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`init-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-s3.yaml)

- [`k8s-json-patch-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-json-patch-workflow.yaml)

- [`k8s-owner-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-owner-reference.yaml)

- [`k8s-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-patch.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`key-only-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/key-only-artifact.yaml)

- [`label-value-from-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/label-value-from-workflow.yaml)

- [`life-cycle-hooks-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-tmpl-level.yaml)

- [`life-cycle-hooks-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-wf-level.yaml)

- [`loops-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-dag.yaml)

- [`loops-maps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-sequence.yaml)

- [`loops.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`node-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/node-selector.yaml)

- [`output-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-gcs.yaml)

- [`output-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-gc-strategy-with-label-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy-with-label-selector.yaml)

- [`pod-gc-strategy.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy.yaml)

- [`pod-metadata-wf-field.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata-wf-field.yaml)

- [`pod-metadata.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-yaml-patch.yaml)

- [`recursive-for-loop.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/recursive-for-loop.yaml)

- [`resubmit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/resubmit.yaml)

- [`retry-backoff.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-backoff.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`retry-container-to-completion.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container.yaml)

- [`retry-on-error.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-on-error.yaml)

- [`retry-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-script.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-python.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)

- [`selected-executor-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/selected-executor-workflow.yaml)

- [`sidecar-dind.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar.yaml)

- [`status-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/status-reference.yaml)

- [`step-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/step-level-timeout.yaml)

- [`steps-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps-inline-workflow.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)

- [`suspend-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template.yaml)

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`synchronization-mutex-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-wf-level.yaml)

- [`template-defaults.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-defaults.yaml)

- [`template-on-exit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-on-exit.yaml)

- [`timeouts-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-pvc.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)

- [`event-consumer-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-event-binding/event-consumer-workflowtemplate.yaml)

- [`workflow-of-workflows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-of-workflows.yaml)

- [`dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/steps.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)

- [`workflow-archive-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-archive-logs.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`workflow-template-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`activeDeadlineSeconds`|`integer`|Optional duration in seconds relative to the workflow start time which the workflow is allowed to run before the controller terminates the io.argoproj.workflow.v1alpha1. A value of zero is used to terminate a Running workflow|
|`affinity`|[`Affinity`](#affinity)|Affinity sets the scheduling constraints for all pods in the io.argoproj.workflow.v1alpha1. Can be overridden by an affinity specified in the template|
|`archiveLogs`|`boolean`|ArchiveLogs indicates if the container logs should be archived|
|`arguments`|[`Arguments`](#arguments)|Arguments contain the parameters and artifacts sent to the workflow entrypoint Parameters are referencable globally using the 'workflow' variable prefix. e.g. {{io.argoproj.workflow.v1alpha1.parameters.myparam}}|
|`artifactRepositoryRef`|[`ArtifactRepositoryRef`](#artifactrepositoryref)|ArtifactRepositoryRef specifies the configMap name and key containing the artifact repository config.|
|`automountServiceAccountToken`|`boolean`|AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods. ServiceAccountName of ExecutorConfig must be specified if this value is false.|
|`dnsConfig`|[`PodDNSConfig`](#poddnsconfig)|PodDNSConfig defines the DNS parameters of a pod in addition to those generated from DNSPolicy.|
|`dnsPolicy`|`string`|Set DNS policy for the pod. Defaults to "ClusterFirst". Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'.|
|`entrypoint`|`string`|Entrypoint is a template reference to the starting point of the io.argoproj.workflow.v1alpha1.|
|`executor`|[`ExecutorConfig`](#executorconfig)|Executor holds configurations of executor containers of the io.argoproj.workflow.v1alpha1.|
|`hooks`|[`LifecycleHook`](#lifecyclehook)|Hooks holds the lifecycle hook which is invoked at lifecycle of step, irrespective of the success, failure, or error status of the primary step|
|`hostAliases`|`Array<`[`HostAlias`](#hostalias)`>`|_No description available_|
|`hostNetwork`|`boolean`|Host networking requested for this workflow pod. Default to false.|
|`imagePullSecrets`|`Array<`[`LocalObjectReference`](#localobjectreference)`>`|ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any images in pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secrets can be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet. More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod|
|`metrics`|[`Metrics`](#metrics)|Metrics are a list of metrics emitted from this Workflow|
|`nodeSelector`|`Map< string , string >`|NodeSelector is a selector which will result in all pods of the workflow to be scheduled on the selected node(s). This is able to be overridden by a nodeSelector specified in the template.|
|`onExit`|`string`|OnExit is a template reference which is invoked at the end of the workflow, irrespective of the success, failure, or error of the primary io.argoproj.workflow.v1alpha1.|
|`parallelism`|`integer`|Parallelism limits the max total parallel pods that can execute at the same time in a workflow|
|`podDisruptionBudget`|[`PodDisruptionBudgetSpec`](#poddisruptionbudgetspec)|PodDisruptionBudget holds the number of concurrent disruptions that you allow for Workflow's Pods. Controller will automatically add the selector with workflow name, if selector is empty. Optional: Defaults to empty.|
|`podGC`|[`PodGC`](#podgc)|PodGC describes the strategy to use when to deleting completed pods|
|`podMetadata`|[`Metadata`](#metadata)|PodMetadata defines additional metadata that should be applied to workflow pods|
|`podPriority`|`integer`|Priority to apply to workflow pods.|
|`podPriorityClassName`|`string`|PriorityClassName to apply to workflow pods.|
|`podSpecPatch`|`string`|PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of container fields which are not strings (e.g. resource limits).|
|`priority`|`integer`|Priority is used if controller is configured to process limited number of workflows in parallel. Workflows with higher priority are processed first.|
|`retryStrategy`|[`RetryStrategy`](#retrystrategy)|RetryStrategy for all templates in the io.argoproj.workflow.v1alpha1.|
|`schedulerName`|`string`|Set scheduler name for all pods. Will be overridden if container/script template's scheduler name is set. Default scheduler will be used if neither specified.|
|`securityContext`|[`PodSecurityContext`](#podsecuritycontext)|SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty.  See type description for default values of each field.|
|`serviceAccountName`|`string`|ServiceAccountName is the name of the ServiceAccount to run all pods of the workflow as.|
|`shutdown`|`string`|Shutdown will shutdown the workflow according to its ShutdownStrategy|
|`suspend`|`boolean`|Suspend will suspend the workflow and prevent execution of any future steps in the workflow|
|`synchronization`|[`Synchronization`](#synchronization)|Synchronization holds synchronization lock configuration for this Workflow|
|`templateDefaults`|[`Template`](#template)|TemplateDefaults holds default template values that will apply to all templates in the Workflow, unless overridden on the template-level|
|`templates`|`Array<`[`Template`](#template)`>`|Templates is a list of workflow templates used in a workflow|
|`tolerations`|`Array<`[`Toleration`](#toleration)`>`|Tolerations to apply to workflow pods.|
|`ttlStrategy`|[`TTLStrategy`](#ttlstrategy)|TTLStrategy limits the lifetime of a Workflow that has finished execution depending on if it Succeeded or Failed. If this struct is set, once the Workflow finishes, it will be deleted after the time to live expires. If this field is unset, the controller config map will hold the default values.|
|`volumeClaimGC`|[`VolumeClaimGC`](#volumeclaimgc)|VolumeClaimGC describes the strategy to use when to deleting volumes from completed workflows|
|`volumeClaimTemplates`|`Array<`[`PersistentVolumeClaim`](#persistentvolumeclaim)`>`|VolumeClaimTemplates is a list of claims that containers are allowed to reference. The Workflow controller will create the claims at the beginning of the workflow and delete the claims upon completion of the workflow|
|`volumes`|`Array<`[`Volume`](#volume)`>`|Volumes is a list of volumes that can be mounted by containers in a io.argoproj.workflow.v1alpha1.|
|`workflowMetadata`|[`WorkflowMetadata`](#workflowmetadata)|WorkflowMetadata contains some metadata of the workflow to be refer|
|`workflowTemplateRef`|[`WorkflowTemplateRef`](#workflowtemplateref)|WorkflowTemplateRef holds a reference to a WorkflowTemplate for execution|

## WorkflowStatus

WorkflowStatus contains overall status information about a workflow

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifactRepositoryRef`|[`ArtifactRepositoryRefStatus`](#artifactrepositoryrefstatus)|ArtifactRepositoryRef is used to cache the repository to use so we do not need to determine it everytime we reconcile.|
|`compressedNodes`|`string`|Compressed and base64 decoded Nodes map|
|`conditions`|`Array<`[`Condition`](#condition)`>`|Conditions is a list of conditions the Workflow may have|
|`estimatedDuration`|`integer`|EstimatedDuration in seconds.|
|`finishedAt`|[`Time`](#time)|Time at which this workflow completed|
|`message`|`string`|A human readable message indicating details about why the workflow is in this condition.|
|`nodes`|[`NodeStatus`](#nodestatus)|Nodes is a mapping between a node ID and the node's status.|
|`offloadNodeStatusVersion`|`string`|Whether on not node status has been offloaded to a database. If exists, then Nodes and CompressedNodes will be empty. This will actually be populated with a hash of the offloaded data.|
|`outputs`|[`Outputs`](#outputs)|Outputs captures output values and artifact locations produced by the workflow via global outputs|
|`persistentVolumeClaims`|`Array<`[`Volume`](#volume)`>`|PersistentVolumeClaims tracks all PVCs that were created as part of the io.argoproj.workflow.v1alpha1. The contents of this list are drained at the end of the workflow.|
|`phase`|`string`|Phase a simple, high-level summary of where the workflow is in its lifecycle.|
|`progress`|`string`|Progress to completion|
|`resourcesDuration`|`Map< integer , int64 >`|ResourcesDuration is the total for the workflow|
|`startedAt`|[`Time`](#time)|Time at which this workflow started|
|`storedTemplates`|[`Template`](#template)|StoredTemplates is a mapping between a template ref and the node's status.|
|`storedWorkflowTemplateSpec`|[`WorkflowSpec`](#workflowspec)|StoredWorkflowSpec stores the WorkflowTemplate spec for future execution.|
|`synchronization`|[`SynchronizationStatus`](#synchronizationstatus)|Synchronization stores the status of synchronization locks|

## CronWorkflowSpec

CronWorkflowSpec is the specification of a CronWorkflow

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-artifacts.yaml)

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`arguments-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`artifact-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`artifact-repository-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-repository-ref.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`workflow-template-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref.yaml)

- [`coinflip-recursive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip.yaml)

- [`colored-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/colored-logs.yaml)

- [`conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-artifacts.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`conditionals-complex.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals-complex.yaml)

- [`conditionals.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals.yaml)

- [`graph-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/graph-workflow.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`parallel-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/parallel-workflow.yaml)

- [`sequence-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/sequence-workflow.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/continue-on-fail.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`cron-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-coinflip.yaml)

- [`dag-conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-artifacts.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`dag-continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-continue-on-fail.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-disable-failFast.yaml)

- [`dag-enhanced-depends.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-enhanced-depends.yaml)

- [`dag-inline-clusterworkflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-clusterworkflowtemplate.yaml)

- [`dag-inline-cronworkflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-cronworkflow.yaml)

- [`dag-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflow.yaml)

- [`dag-inline-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflowtemplate.yaml)

- [`dag-multiroot.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-nested.yaml)

- [`dag-targets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-targets.yaml)

- [`dag-task-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-task-level-timeout.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`default-pdb-support.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/default-pdb-support.yaml)

- [`dns-config.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dns-config.yaml)

- [`exit-code-output-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-code-output-variable.yaml)

- [`exit-handler-dag-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-dag-level.yaml)

- [`exit-handler-slack.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-slack.yaml)

- [`exit-handler-step-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-step-level.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`exit-handlers.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handlers.yaml)

- [`expression-destructure-json.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-destructure-json.yaml)

- [`expression-reusing-verbose-snippets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-reusing-verbose-snippets.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`forever.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/forever.yaml)

- [`fun-with-gifs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/gc-ttl.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)

- [`global-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)

- [`hello-hybrid.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-hybrid.yaml)

- [`hello-windows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-windows.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-world.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`http-success-condition.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-success-condition.yaml)

- [`image-pull-secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`init-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-s3.yaml)

- [`k8s-json-patch-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-json-patch-workflow.yaml)

- [`k8s-owner-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-owner-reference.yaml)

- [`k8s-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-patch.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`key-only-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/key-only-artifact.yaml)

- [`label-value-from-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/label-value-from-workflow.yaml)

- [`life-cycle-hooks-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-tmpl-level.yaml)

- [`life-cycle-hooks-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-wf-level.yaml)

- [`loops-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-dag.yaml)

- [`loops-maps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-sequence.yaml)

- [`loops.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`node-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/node-selector.yaml)

- [`output-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-gcs.yaml)

- [`output-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-gc-strategy-with-label-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy-with-label-selector.yaml)

- [`pod-gc-strategy.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy.yaml)

- [`pod-metadata-wf-field.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata-wf-field.yaml)

- [`pod-metadata.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-yaml-patch.yaml)

- [`recursive-for-loop.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/recursive-for-loop.yaml)

- [`resubmit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/resubmit.yaml)

- [`retry-backoff.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-backoff.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`retry-container-to-completion.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container.yaml)

- [`retry-on-error.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-on-error.yaml)

- [`retry-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-script.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-python.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)

- [`selected-executor-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/selected-executor-workflow.yaml)

- [`sidecar-dind.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar.yaml)

- [`status-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/status-reference.yaml)

- [`step-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/step-level-timeout.yaml)

- [`steps-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps-inline-workflow.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)

- [`suspend-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template.yaml)

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`synchronization-mutex-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-wf-level.yaml)

- [`template-defaults.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-defaults.yaml)

- [`template-on-exit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-on-exit.yaml)

- [`timeouts-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-pvc.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)

- [`event-consumer-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-event-binding/event-consumer-workflowtemplate.yaml)

- [`workflow-of-workflows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-of-workflows.yaml)

- [`dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/steps.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)

- [`workflow-archive-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-archive-logs.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`workflow-template-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`concurrencyPolicy`|`string`|ConcurrencyPolicy is the K8s-style concurrency policy that will be used|
|`failedJobsHistoryLimit`|`integer`|FailedJobsHistoryLimit is the number of failed jobs to be kept at a time|
|`schedule`|`string`|Schedule is a schedule to run the Workflow in Cron format|
|`startingDeadlineSeconds`|`integer`|StartingDeadlineSeconds is the K8s-style deadline that will limit the time a CronWorkflow will be run after its original scheduled time if it is missed.|
|`successfulJobsHistoryLimit`|`integer`|SuccessfulJobsHistoryLimit is the number of successful jobs to be kept at a time|
|`suspend`|`boolean`|Suspend is a flag that will stop new CronWorkflows from running if set to true|
|`timezone`|`string`|Timezone is the timezone against which the cron schedule will be calculated, e.g. "Asia/Tokyo". Default is machine's local time.|
|`workflowMetadata`|[`ObjectMeta`](#objectmeta)|WorkflowMetadata contains some metadata of the workflow to be run|
|`workflowSpec`|[`WorkflowSpec`](#workflowspec)|WorkflowSpec is the spec of the workflow to be run|

## CronWorkflowStatus

CronWorkflowStatus is the status of a CronWorkflow

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`active`|`Array<`[`ObjectReference`](#objectreference)`>`|Active is a list of active workflows stemming from this CronWorkflow|
|`conditions`|`Array<`[`Condition`](#condition)`>`|Conditions is a list of conditions the CronWorkflow may have|
|`lastScheduledTime`|[`Time`](#time)|LastScheduleTime is the last time the CronWorkflow was scheduled|

## Arguments

Arguments to a template

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-artifacts.yaml)

- [`arguments-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`artifact-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`conditionals.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond.yaml)

- [`dag-multiroot.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-nested.yaml)

- [`dag-targets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-targets.yaml)

- [`dag-task-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-task-level-timeout.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`exit-code-output-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-code-output-variable.yaml)

- [`exit-handler-dag-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-dag-level.yaml)

- [`exit-handler-step-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-step-level.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`expression-destructure-json.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-destructure-json.yaml)

- [`expression-reusing-verbose-snippets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-reusing-verbose-snippets.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)

- [`global-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`http-success-condition.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-success-condition.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`label-value-from-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/label-value-from-workflow.yaml)

- [`loops-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-dag.yaml)

- [`loops-maps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-sequence.yaml)

- [`loops.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`node-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/node-selector.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-metadata-wf-field.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata-wf-field.yaml)

- [`pod-metadata.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-yaml-patch.yaml)

- [`recursive-for-loop.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/recursive-for-loop.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`scripts-bash.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-python.yaml)

- [`step-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/step-level-timeout.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)

- [`event-consumer-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-event-binding/event-consumer-workflowtemplate.yaml)

- [`workflow-of-workflows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-of-workflows.yaml)

- [`dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/hello-world.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/steps.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifacts`|`Array<`[`Artifact`](#artifact)`>`|Artifacts is the list of artifacts to pass to the template or workflow|
|`parameters`|`Array<`[`Parameter`](#parameter)`>`|Parameters is the list of parameters to pass to the template or workflow|

## ArtifactRepositoryRef

_No description available_

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-repository-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-repository-ref.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`configMap`|`string`|The name of the config map. Defaults to "artifact-repositories".|
|`key`|`string`|The config map key. Defaults to the value of the "workflows.argoproj.io/default-artifact-repository" annotation.|

## ExecutorConfig

ExecutorConfig holds configurations of an executor container.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`graph-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/graph-workflow.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`sequence-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/sequence-workflow.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`selected-executor-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/selected-executor-workflow.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`serviceAccountName`|`string`|ServiceAccountName specifies the service account name of the executor container.|

## LifecycleHook

_No description available_

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`life-cycle-hooks-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-tmpl-level.yaml)

- [`life-cycle-hooks-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-wf-level.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`arguments`|[`Arguments`](#arguments)|Arguments hold arguments to the template|
|`expression`|`string`|Expression is a condition expression for when a node will be retried. If it evaluates to false, the node will not be retried and the retry strategy will be ignored|
|`template`|`string`|Template is the name of the template to execute by the hook|
|`templateRef`|[`TemplateRef`](#templateref)|TemplateRef is the reference to the template resource to execute by the hook|

## Metrics

Metrics are a list of metrics emitted from a Workflow/Template

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`prometheus`|`Array<`[`Prometheus`](#prometheus)`>`|Prometheus is a list of prometheus metrics to be emitted|

## PodGC

PodGC describes how to delete completed pods as they complete

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`pod-gc-strategy-with-label-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy-with-label-selector.yaml)

- [`pod-gc-strategy.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`labelSelector`|[`LabelSelector`](#labelselector)|LabelSelector is the label selector to check if the pods match the labels before being added to the pod GC queue.|
|`strategy`|`string`|Strategy is the strategy to use. One of "OnPodCompletion", "OnPodSuccess", "OnWorkflowCompletion", "OnWorkflowSuccess"|

## Metadata

Pod metdata

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`pod-metadata-wf-field.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata-wf-field.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`annotations`|`Map< string , string >`|_No description available_|
|`labels`|`Map< string , string >`|_No description available_|

## RetryStrategy

RetryStrategy provides controls on how to retry a workflow step

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`dag-disable-failFast.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-disable-failFast.yaml)

- [`retry-backoff.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-backoff.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`retry-container-to-completion.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container.yaml)

- [`retry-on-error.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-on-error.yaml)

- [`retry-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-script.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-with-steps.yaml)

- [`template-defaults.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-defaults.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`affinity`|[`RetryAffinity`](#retryaffinity)|Affinity prevents running workflow's step on the same host|
|`backoff`|[`Backoff`](#backoff)|Backoff is a backoff strategy|
|`expression`|`string`|Expression is a condition expression for when a node will be retried. If it evaluates to false, the node will not be retried and the retry strategy will be ignored|
|`limit`|[`IntOrString`](#intorstring)|Limit is the maximum number of attempts when retrying a container|
|`retryPolicy`|`string`|RetryPolicy is a policy of NodePhase statuses that will be retried|

## Synchronization

Synchronization holds synchronization lock configuration

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`synchronization-mutex-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-wf-level.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`mutex`|[`Mutex`](#mutex)|Mutex holds the Mutex lock details|
|`semaphore`|[`SemaphoreRef`](#semaphoreref)|Semaphore holds the Semaphore configuration|

## Template

Template is a reusable and composable unit of execution in a workflow

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`template-defaults.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-defaults.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`activeDeadlineSeconds`|[`IntOrString`](#intorstring)|Optional duration in seconds relative to the StartTime that the pod may be active on a node before the system actively tries to terminate the pod; value must be positive integer This field is only applicable to container and script templates.|
|`affinity`|[`Affinity`](#affinity)|Affinity sets the pod's scheduling constraints Overrides the affinity set at the workflow level (if any)|
|`archiveLocation`|[`ArtifactLocation`](#artifactlocation)|Location in which all files related to the step will be stored (logs, artifacts, etc...). Can be overridden by individual items in Outputs. If omitted, will use the default artifact repository location configured in the controller, appended with the <workflowname>/<nodename> in the key.|
|`automountServiceAccountToken`|`boolean`|AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods. ServiceAccountName of ExecutorConfig must be specified if this value is false.|
|`container`|[`Container`](#container)|Container is the main container image to run in the pod|
|`containerSet`|[`ContainerSetTemplate`](#containersettemplate)|ContainerSet groups multiple containers within a single pod.|
|`daemon`|`boolean`|Deamon will allow a workflow to proceed to the next step so long as the container reaches readiness|
|`dag`|[`DAGTemplate`](#dagtemplate)|DAG template subtype which runs a DAG|
|`data`|[`Data`](#data)|Data is a data template|
|`executor`|[`ExecutorConfig`](#executorconfig)|Executor holds configurations of the executor container.|
|`failFast`|`boolean`|FailFast, if specified, will fail this template if any of its child pods has failed. This is useful for when this template is expanded with `withItems`, etc.|
|`hostAliases`|`Array<`[`HostAlias`](#hostalias)`>`|HostAliases is an optional list of hosts and IPs that will be injected into the pod spec|
|`http`|[`HTTP`](#http)|HTTP makes a HTTP request|
|`initContainers`|`Array<`[`UserContainer`](#usercontainer)`>`|InitContainers is a list of containers which run before the main container.|
|`inputs`|[`Inputs`](#inputs)|Inputs describe what inputs parameters and artifacts are supplied to this template|
|`memoize`|[`Memoize`](#memoize)|Memoize allows templates to use outputs generated from already executed templates|
|`metadata`|[`Metadata`](#metadata)|Metdata sets the pods's metadata, i.e. annotations and labels|
|`metrics`|[`Metrics`](#metrics)|Metrics are a list of metrics emitted from this template|
|`name`|`string`|Name is the name of the template|
|`nodeSelector`|`Map< string , string >`|NodeSelector is a selector to schedule this step of the workflow to be run on the selected node(s). Overrides the selector set at the workflow level.|
|`outputs`|[`Outputs`](#outputs)|Outputs describe the parameters and artifacts that this template produces|
|`parallelism`|`integer`|Parallelism limits the max total parallel pods that can execute at the same time within the boundaries of this template invocation. If additional steps/dag templates are invoked, the pods created by those templates will not be counted towards this total.|
|`plugin`|[`Plugin`](#plugin)|Plugin is a plugin template|
|`podSpecPatch`|`string`|PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of container fields which are not strings (e.g. resource limits).|
|`priority`|`integer`|Priority to apply to workflow pods.|
|`priorityClassName`|`string`|PriorityClassName to apply to workflow pods.|
|`resource`|[`ResourceTemplate`](#resourcetemplate)|Resource template subtype which can run k8s resources|
|`retryStrategy`|[`RetryStrategy`](#retrystrategy)|RetryStrategy describes how to retry a template when it fails|
|`schedulerName`|`string`|If specified, the pod will be dispatched by specified scheduler. Or it will be dispatched by workflow scope scheduler if specified. If neither specified, the pod will be dispatched by default scheduler.|
|`script`|[`ScriptTemplate`](#scripttemplate)|Script runs a portion of code against an interpreter|
|`securityContext`|[`PodSecurityContext`](#podsecuritycontext)|SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty.  See type description for default values of each field.|
|`serviceAccountName`|`string`|ServiceAccountName to apply to workflow pods|
|`sidecars`|`Array<`[`UserContainer`](#usercontainer)`>`|Sidecars is a list of containers which run alongside the main container Sidecars are automatically killed when the main container completes|
|`steps`|`Array<Array<`[`WorkflowStep`](#workflowstep)`>>`|Steps define a series of sequential/parallel workflow steps|
|`suspend`|[`SuspendTemplate`](#suspendtemplate)|Suspend template subtype which can suspend a workflow when reaching the step|
|`synchronization`|[`Synchronization`](#synchronization)|Synchronization holds synchronization lock configuration for this template|
|`timeout`|`string`|Timeout allows to set the total node execution timeout duration counting from the node's start time. This duration also includes time in which the node spends in Pending state. This duration may not be applied to Step or DAG templates.|
|`tolerations`|`Array<`[`Toleration`](#toleration)`>`|Tolerations to apply to workflow pods.|
|`volumes`|`Array<`[`Volume`](#volume)`>`|Volumes is a list of volumes that can be mounted by containers in a template.|

## TTLStrategy

TTLStrategy is the strategy for the time to live depending on if the workflow succeeded or failed

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`gc-ttl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/gc-ttl.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`secondsAfterCompletion`|`integer`|SecondsAfterCompletion is the number of seconds to live after completion|
|`secondsAfterFailure`|`integer`|SecondsAfterFailure is the number of seconds to live after failure|
|`secondsAfterSuccess`|`integer`|SecondsAfterSuccess is the number of seconds to live after success|

## VolumeClaimGC

VolumeClaimGC describes how to delete volumes from completed Workflows

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`strategy`|`string`|Strategy is the strategy to use. One of "OnWorkflowCompletion", "OnWorkflowSuccess"|

## WorkflowMetadata

_No description available_

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`label-value-from-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/label-value-from-workflow.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`annotations`|`Map< string , string >`|_No description available_|
|`labels`|`Map< string , string >`|_No description available_|
|`labelsFrom`|[`LabelValueFrom`](#labelvaluefrom)|_No description available_|

## WorkflowTemplateRef

WorkflowTemplateRef is a reference to a WorkflowTemplate resource.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`workflow-template-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`dag-inline-cronworkflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-cronworkflow.yaml)

- [`workflow-of-workflows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-of-workflows.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`workflow-template-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`clusterScope`|`boolean`|ClusterScope indicates the referred template is cluster scoped (i.e. a ClusterWorkflowTemplate).|
|`name`|`string`|Name is the resource name of the workflow template.|

## ArtifactRepositoryRefStatus

_No description available_

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-repository-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-repository-ref.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifactRepository`|[`ArtifactRepository`](#artifactrepository)|The repository the workflow will use. This maybe empty before v3.1.|
|`configMap`|`string`|The name of the config map. Defaults to "artifact-repositories".|
|`default`|`boolean`|If this ref represents the default artifact repository, rather than a config map.|
|`key`|`string`|The config map key. Defaults to the value of the "workflows.argoproj.io/default-artifact-repository" annotation.|
|`namespace`|`string`|The namespace of the config map. Defaults to the workflow's namespace, or the controller's namespace (if found).|

## Condition

_No description available_

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`message`|`string`|Message is the condition message|
|`status`|`string`|Status is the status of the condition|
|`type`|`string`|Type is the type of condition|

## NodeStatus

NodeStatus contains status information about an individual node in the workflow

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`boundaryID`|`string`|BoundaryID indicates the node ID of the associated template root node in which this node belongs to|
|`children`|`Array< string >`|Children is a list of child node IDs|
|`daemoned`|`boolean`|Daemoned tracks whether or not this node was daemoned and need to be terminated|
|`displayName`|`string`|DisplayName is a human readable representation of the node. Unique within a template boundary|
|`estimatedDuration`|`integer`|EstimatedDuration in seconds.|
|`finishedAt`|[`Time`](#time)|Time at which this node completed|
|`hostNodeName`|`string`|HostNodeName name of the Kubernetes node on which the Pod is running, if applicable|
|`id`|`string`|ID is a unique identifier of a node within the worklow It is implemented as a hash of the node name, which makes the ID deterministic|
|`inputs`|[`Inputs`](#inputs)|Inputs captures input parameter values and artifact locations supplied to this template invocation|
|`memoizationStatus`|[`MemoizationStatus`](#memoizationstatus)|MemoizationStatus holds information about cached nodes|
|`message`|`string`|A human readable message indicating details about why the node is in this condition.|
|`name`|`string`|Name is unique name in the node tree used to generate the node ID|
|`outboundNodes`|`Array< string >`|OutboundNodes tracks the node IDs which are considered "outbound" nodes to a template invocation. For every invocation of a template, there are nodes which we considered as "outbound". Essentially, these are last nodes in the execution sequence to run, before the template is considered completed. These nodes are then connected as parents to a following step.In the case of single pod steps (i.e. container, script, resource templates), this list will be nil since the pod itself is already considered the "outbound" node. In the case of DAGs, outbound nodes are the "target" tasks (tasks with no children). In the case of steps, outbound nodes are all the containers involved in the last step group. NOTE: since templates are composable, the list of outbound nodes are carried upwards when a DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes of a template, will be a superset of the outbound nodes of its last children.|
|`outputs`|[`Outputs`](#outputs)|Outputs captures output parameter values and artifact locations produced by this template invocation|
|`phase`|`string`|Phase a simple, high-level summary of where the node is in its lifecycle. Can be used as a state machine.|
|`podIP`|`string`|PodIP captures the IP of the pod for daemoned steps|
|`progress`|`string`|Progress to completion|
|`resourcesDuration`|`Map< integer , int64 >`|ResourcesDuration is indicative, but not accurate, resource duration. This is populated when the nodes completes.|
|`startedAt`|[`Time`](#time)|Time at which this node started|
|`synchronizationStatus`|[`NodeSynchronizationStatus`](#nodesynchronizationstatus)|SynchronizationStatus is the synchronization status of the node|
|`templateName`|`string`|TemplateName is the template name which this node corresponds to. Not applicable to virtual nodes (e.g. Retry, StepGroup)|
|`templateRef`|[`TemplateRef`](#templateref)|TemplateRef is the reference to the template resource which this node corresponds to. Not applicable to virtual nodes (e.g. Retry, StepGroup)|
|`templateScope`|`string`|TemplateScope is the template scope in which the template of this node was retrieved.|
|`type`|`string`|Type indicates type of node|

## Outputs

Outputs hold parameters, artifacts, and results from a step

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`artifact-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`artifact-repository-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-repository-ref.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-artifacts.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`dag-conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-artifacts.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`fun-with-gifs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fun-with-gifs.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`key-only-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/key-only-artifact.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`output-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-gcs.yaml)

- [`output-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifacts`|`Array<`[`Artifact`](#artifact)`>`|Artifacts holds the list of output artifacts produced by a step|
|`exitCode`|`string`|ExitCode holds the exit code of a script template|
|`parameters`|`Array<`[`Parameter`](#parameter)`>`|Parameters holds the list of output parameters produced by a step|
|`result`|`string`|Result holds the result (stdout) of a script template|

## SynchronizationStatus

SynchronizationStatus stores the status of semaphore and mutex.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`synchronization-mutex-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-wf-level.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`mutex`|[`MutexStatus`](#mutexstatus)|Mutex stores this workflow's mutex holder details|
|`semaphore`|[`SemaphoreStatus`](#semaphorestatus)|Semaphore stores this workflow's Semaphore holder details|

## Artifact

Artifact indicates an artifact to place at a specified path

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-artifacts.yaml)

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`artifact-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`artifact-repository-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-repository-ref.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-artifacts.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`dag-conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-artifacts.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`fun-with-gifs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fun-with-gifs.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`input-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-s3.yaml)

- [`key-only-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/key-only-artifact.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`output-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-gcs.yaml)

- [`output-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-s3.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`archive`|[`ArchiveStrategy`](#archivestrategy)|Archive controls how the artifact will be saved to the artifact repository.|
|`archiveLogs`|`boolean`|ArchiveLogs indicates if the container logs should be archived|
|`artifactory`|[`ArtifactoryArtifact`](#artifactoryartifact)|Artifactory contains artifactory artifact location details|
|`from`|`string`|From allows an artifact to reference an artifact from a previous step|
|`fromExpression`|`string`|FromExpression, if defined, is evaluated to specify the value for the artifact|
|`gcs`|[`GCSArtifact`](#gcsartifact)|GCS contains GCS artifact location details|
|`git`|[`GitArtifact`](#gitartifact)|Git contains git artifact location details|
|`globalName`|`string`|GlobalName exports an output artifact to the global scope, making it available as '{{io.argoproj.workflow.v1alpha1.outputs.artifacts.XXXX}} and in workflow.status.outputs.artifacts|
|`hdfs`|[`HDFSArtifact`](#hdfsartifact)|HDFS contains HDFS artifact location details|
|`http`|[`HTTPArtifact`](#httpartifact)|HTTP contains HTTP artifact location details|
|`mode`|`integer`|mode bits to use on this file, must be a value between 0 and 0777 set when loading input artifacts.|
|`name`|`string`|name of the artifact. must be unique within a template's inputs/outputs.|
|`optional`|`boolean`|Make Artifacts optional, if Artifacts doesn't generate or exist|
|`oss`|[`OSSArtifact`](#ossartifact)|OSS contains OSS artifact location details|
|`path`|`string`|Path is the container path to the artifact|
|`raw`|[`RawArtifact`](#rawartifact)|Raw contains raw artifact location details|
|`recurseMode`|`boolean`|If mode is set, apply the permission recursively into the artifact if it is a folder|
|`s3`|[`S3Artifact`](#s3artifact)|S3 contains S3 artifact location details|
|`subPath`|`string`|SubPath allows an artifact to be sourced from a subpath within the specified source|

## Parameter

Parameter indicate a passed string parameter to a service template with an optional default value

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`arguments-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`conditionals.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond.yaml)

- [`dag-multiroot.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-nested.yaml)

- [`dag-targets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-targets.yaml)

- [`dag-task-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-task-level-timeout.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`exit-code-output-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-code-output-variable.yaml)

- [`exit-handler-dag-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-dag-level.yaml)

- [`exit-handler-step-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-step-level.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`expression-destructure-json.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-destructure-json.yaml)

- [`expression-reusing-verbose-snippets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-reusing-verbose-snippets.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)

- [`global-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`http-success-condition.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-success-condition.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`label-value-from-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/label-value-from-workflow.yaml)

- [`loops-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-dag.yaml)

- [`loops-maps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-sequence.yaml)

- [`loops.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`node-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/node-selector.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-metadata-wf-field.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata-wf-field.yaml)

- [`pod-metadata.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-yaml-patch.yaml)

- [`recursive-for-loop.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/recursive-for-loop.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`scripts-bash.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-python.yaml)

- [`step-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/step-level-timeout.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)

- [`event-consumer-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-event-binding/event-consumer-workflowtemplate.yaml)

- [`workflow-of-workflows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-of-workflows.yaml)

- [`dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/hello-world.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/steps.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`default`|`string`|Default is the default value to use for an input parameter if a value was not supplied|
|`description`|`string`|Description is the parameter description|
|`enum`|`Array< string >`|Enum holds a list of string values to choose from, for the actual value of the parameter|
|`globalName`|`string`|GlobalName exports an output parameter to the global scope, making it available as '{{io.argoproj.workflow.v1alpha1.outputs.parameters.XXXX}} and in workflow.status.outputs.parameters|
|`name`|`string`|Name is the parameter name|
|`value`|`string`|Value is the literal value to use for the parameter. If specified in the context of an input parameter, the value takes precedence over any passed values|
|`valueFrom`|[`ValueFrom`](#valuefrom)|ValueFrom is the source for the output parameter's value|

## TemplateRef

TemplateRef is a reference of template resource.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`cluster-wftmpl-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/steps.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`clusterScope`|`boolean`|ClusterScope indicates the referred template is cluster scoped (i.e. a ClusterWorkflowTemplate).|
|`name`|`string`|Name is the resource name of the template.|
|`template`|`string`|Template is the name of referred template in the resource.|

## Prometheus

Prometheus is a prometheus metric to be emitted

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`counter`|[`Counter`](#counter)|Counter is a counter metric|
|`gauge`|[`Gauge`](#gauge)|Gauge is a gauge metric|
|`help`|`string`|Help is a string that describes the metric|
|`histogram`|[`Histogram`](#histogram)|Histogram is a histogram metric|
|`labels`|`Array<`[`MetricLabel`](#metriclabel)`>`|Labels is a list of metric labels|
|`name`|`string`|Name is the name of the metric|
|`when`|`string`|When is a conditional statement that decides when to emit the metric|

## RetryAffinity

RetryAffinity prevents running steps on the same host.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`nodeAntiAffinity`|[`RetryNodeAntiAffinity`](#retrynodeantiaffinity)|_No description available_|

## Backoff

Backoff is a backoff strategy to use within retryStrategy

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`retry-backoff.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-backoff.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`duration`|`string`|Duration is the amount to back off. Default unit is seconds, but could also be a duration (e.g. "2m", "1h")|
|`factor`|[`IntOrString`](#intorstring)|Factor is a factor to multiply the base duration after each failed retry|
|`maxDuration`|`string`|MaxDuration is the maximum amount of time allowed for the backoff strategy|

## Mutex

Mutex holds Mutex configuration

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`synchronization-mutex-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-wf-level.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|name of the mutex|

## SemaphoreRef

SemaphoreRef is a reference of Semaphore

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`configMapKeyRef`|[`ConfigMapKeySelector`](#configmapkeyselector)|ConfigMapKeyRef is configmap selector for Semaphore configuration|

## ArtifactLocation

ArtifactLocation describes a location for a single or multiple artifacts. It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname). It is also used to describe the location of multiple artifacts such as the archive location of a single workflow step, which the executor will use as a default location to store its files.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/archive-location.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`archiveLogs`|`boolean`|ArchiveLogs indicates if the container logs should be archived|
|`artifactory`|[`ArtifactoryArtifact`](#artifactoryartifact)|Artifactory contains artifactory artifact location details|
|`gcs`|[`GCSArtifact`](#gcsartifact)|GCS contains GCS artifact location details|
|`git`|[`GitArtifact`](#gitartifact)|Git contains git artifact location details|
|`hdfs`|[`HDFSArtifact`](#hdfsartifact)|HDFS contains HDFS artifact location details|
|`http`|[`HTTPArtifact`](#httpartifact)|HTTP contains HTTP artifact location details|
|`oss`|[`OSSArtifact`](#ossartifact)|OSS contains OSS artifact location details|
|`raw`|[`RawArtifact`](#rawartifact)|Raw contains raw artifact location details|
|`s3`|[`S3Artifact`](#s3artifact)|S3 contains S3 artifact location details|

## ContainerSetTemplate

_No description available_

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`graph-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/graph-workflow.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`parallel-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/parallel-workflow.yaml)

- [`sequence-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/sequence-workflow.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`containers`|`Array<`[`ContainerNode`](#containernode)`>`|_No description available_|
|`retryStrategy`|[`ContainerSetRetryStrategy`](#containersetretrystrategy)|RetryStrategy describes how to retry a container nodes in the container set if it fails. Nbr of retries(default 0) and sleep duration between retries(default 0s, instant retry) can be set.|
|`volumeMounts`|`Array<`[`VolumeMount`](#volumemount)`>`|_No description available_|

## DAGTemplate

DAGTemplate is a template subtype for directed acyclic graph templates

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`cluster-wftmpl-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`dag-coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-coinflip.yaml)

- [`dag-conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-artifacts.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`dag-continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-continue-on-fail.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-disable-failFast.yaml)

- [`dag-enhanced-depends.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-enhanced-depends.yaml)

- [`dag-inline-clusterworkflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-clusterworkflowtemplate.yaml)

- [`dag-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflow.yaml)

- [`dag-inline-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflowtemplate.yaml)

- [`dag-multiroot.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-nested.yaml)

- [`dag-targets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-targets.yaml)

- [`dag-task-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-task-level-timeout.yaml)

- [`exit-handler-dag-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-dag-level.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`key-only-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/key-only-artifact.yaml)

- [`loops-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-dag.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-dag.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`resubmit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/resubmit.yaml)

- [`dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/dag.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`failFast`|`boolean`|This flag is for DAG logic. The DAG logic has a built-in "fail fast" feature to stop scheduling new steps, as soon as it detects that one of the DAG nodes is failed. Then it waits until all DAG nodes are completed before failing the DAG itself. The FailFast flag default is true,  if set to false, it will allow a DAG to run all branches of the DAG to completion (either success or failure), regardless of the failed outcomes of branches in the DAG. More info and example about this feature at https://github.com/argoproj/argo-workflows/issues/1442|
|`target`|`string`|Target are one or more names of targets to execute in a DAG|
|`tasks`|`Array<`[`DAGTask`](#dagtask)`>`|Tasks are a list of DAG tasks|

## Data

Data is a data template

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`input-artifact-raw.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-raw.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`source`|[`DataSource`](#datasource)|Source sources external data into a data template|
|`transformation`|`Array<`[`TransformationStep`](#transformationstep)`>`|Transformation applies a set of transformations|

## HTTP

_No description available_

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-artifacts.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`http-success-condition.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-success-condition.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`input-artifact-http.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-oss.yaml)

- [`life-cycle-hooks-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-tmpl-level.yaml)

- [`life-cycle-hooks-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-wf-level.yaml)

- [`sidecar-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`body`|`string`|Body is content of the HTTP Request|
|`headers`|`Array<`[`HTTPHeader`](#httpheader)`>`|Headers are an optional list of headers to send with HTTP requests|
|`insecureSkipVerify`|`boolean`|insecureSkipVerify is a bool when if set to true will skip TLS verification for the HTTP client|
|`method`|`string`|Method is HTTP methods for HTTP Request|
|`successCondition`|`string`|SuccessCondition is an expression if evaluated to true is considered successful|
|`timeoutSeconds`|`integer`|TimeoutSeconds is request timeout for HTTP Request. Default is 30 seconds|
|`url`|`string`|URL of the HTTP Request|

## UserContainer

UserContainer is a container specified by a user.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`init-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/init-container.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`args`|`Array< string >`|Arguments to the entrypoint. The docker image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell|
|`command`|`Array< string >`|Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell|
|`env`|`Array<`[`EnvVar`](#envvar)`>`|List of environment variables to set in the container. Cannot be updated.|
|`envFrom`|`Array<`[`EnvFromSource`](#envfromsource)`>`|List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.|
|`image`|`string`|Docker image name. More info: https://kubernetes.io/docs/concepts/containers/images This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets.|
|`imagePullPolicy`|`string`|Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images|
|`lifecycle`|[`Lifecycle`](#lifecycle)|Actions that the management system should take in response to container lifecycle events. Cannot be updated.|
|`livenessProbe`|[`Probe`](#probe)|Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`mirrorVolumeMounts`|`boolean`|MirrorVolumeMounts will mount the same volumes specified in the main container to the container (including artifacts), at the same mountPaths. This enables dind daemon to partially see the same filesystem as the main container in order to use features such as docker volume binding|
|`name`|`string`|Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.|
|`ports`|`Array<`[`ContainerPort`](#containerport)`>`|List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated.|
|`readinessProbe`|[`Probe`](#probe)|Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`resources`|[`ResourceRequirements`](#resourcerequirements)|Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/|
|`securityContext`|[`SecurityContext`](#securitycontext)|SecurityContext defines the security options the container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext. More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/|
|`startupProbe`|[`Probe`](#probe)|StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully. If this probe fails, the Pod will be restarted, just as if the livenessProbe failed. This can be used to provide different probe parameters at the beginning of a Pod's lifecycle, when it might take a long time to load data or warm a cache, than during steady-state operation. This cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`stdin`|`boolean`|Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false.|
|`stdinOnce`|`boolean`|Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false|
|`terminationMessagePath`|`string`|Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated.|
|`terminationMessagePolicy`|`string`|Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated.|
|`tty`|`boolean`|Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false.|
|`volumeDevices`|`Array<`[`VolumeDevice`](#volumedevice)`>`|volumeDevices is the list of block devices to be used by the container.|
|`volumeMounts`|`Array<`[`VolumeMount`](#volumemount)`>`|Pod volumes to mount into the container's filesystem. Cannot be updated.|
|`workingDir`|`string`|Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.|

## Inputs

Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-artifacts.yaml)

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`arguments-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`artifact-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`conditionals.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond.yaml)

- [`dag-multiroot.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-nested.yaml)

- [`dag-targets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-targets.yaml)

- [`dag-task-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-task-level-timeout.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`exit-code-output-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-code-output-variable.yaml)

- [`exit-handler-dag-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-dag-level.yaml)

- [`exit-handler-step-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-step-level.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`expression-destructure-json.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-destructure-json.yaml)

- [`expression-reusing-verbose-snippets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-reusing-verbose-snippets.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`http-success-condition.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-success-condition.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`input-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-s3.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`key-only-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/key-only-artifact.yaml)

- [`loops-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-dag.yaml)

- [`loops-maps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-sequence.yaml)

- [`loops.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`node-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/node-selector.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-metadata-wf-field.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata-wf-field.yaml)

- [`pod-metadata.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`recursive-for-loop.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/recursive-for-loop.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`scripts-bash.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-python.yaml)

- [`step-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/step-level-timeout.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)

- [`event-consumer-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-event-binding/event-consumer-workflowtemplate.yaml)

- [`workflow-of-workflows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-of-workflows.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifacts`|`Array<`[`Artifact`](#artifact)`>`|Artifact are a list of artifacts passed as inputs|
|`parameters`|`Array<`[`Parameter`](#parameter)`>`|Parameters are a list of parameters passed as inputs|

## Memoize

Memoization enables caching for the Outputs of the template

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`cache`|[`Cache`](#cache)|Cache sets and configures the kind of cache|
|`key`|`string`|Key is the key to use as the caching key|
|`maxAge`|`string`|MaxAge is the maximum age (e.g. "180s", "24h") of an entry that is still considered valid. If an entry is older than the MaxAge, it will be ignored.|

## Plugin

Plugin is an Object with exactly one key

## ResourceTemplate

ResourceTemplate is a template subtype to manipulate kubernetes resources

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`k8s-json-patch-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-json-patch-workflow.yaml)

- [`k8s-owner-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-owner-reference.yaml)

- [`k8s-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-patch.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`workflow-of-workflows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-of-workflows.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`action`|`string`|Action is the action to perform to the resource. Must be one of: get, create, apply, delete, replace, patch|
|`failureCondition`|`string`|FailureCondition is a label selector expression which describes the conditions of the k8s resource in which the step was considered failed|
|`flags`|`Array< string >`|Flags is a set of additional options passed to kubectl before submitting a resource I.e. to disable resource validation: flags: [	"--validate=false"  # disable resource validation]|
|`manifest`|`string`|Manifest contains the kubernetes manifest|
|`mergeStrategy`|`string`|MergeStrategy is the strategy used to merge a patch. It defaults to "strategic" Must be one of: strategic, merge, json|
|`setOwnerReference`|`boolean`|SetOwnerReference sets the reference to the workflow on the OwnerReference of generated resource.|
|`successCondition`|`string`|SuccessCondition is a label selector expression which describes the conditions of the k8s resource in which it is acceptable to proceed to the following step|

## ScriptTemplate

ScriptTemplate is a template subtype to enable scripting through code steps

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`coinflip-recursive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip.yaml)

- [`colored-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/colored-logs.yaml)

- [`conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-artifacts.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`conditionals-complex.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals-complex.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`dag-coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-coinflip.yaml)

- [`dag-conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-artifacts.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`expression-destructure-json.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-destructure-json.yaml)

- [`expression-reusing-verbose-snippets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-reusing-verbose-snippets.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`loops-param-result.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-result.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`parameter-aggregation-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`recursive-for-loop.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/recursive-for-loop.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`retry-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-script.yaml)

- [`scripts-bash.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-python.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`args`|`Array< string >`|Arguments to the entrypoint. The docker image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell|
|`command`|`Array< string >`|Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell|
|`env`|`Array<`[`EnvVar`](#envvar)`>`|List of environment variables to set in the container. Cannot be updated.|
|`envFrom`|`Array<`[`EnvFromSource`](#envfromsource)`>`|List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.|
|`image`|`string`|Docker image name. More info: https://kubernetes.io/docs/concepts/containers/images This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets.|
|`imagePullPolicy`|`string`|Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images|
|`lifecycle`|[`Lifecycle`](#lifecycle)|Actions that the management system should take in response to container lifecycle events. Cannot be updated.|
|`livenessProbe`|[`Probe`](#probe)|Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`name`|`string`|Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.|
|`ports`|`Array<`[`ContainerPort`](#containerport)`>`|List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated.|
|`readinessProbe`|[`Probe`](#probe)|Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`resources`|[`ResourceRequirements`](#resourcerequirements)|Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/|
|`securityContext`|[`SecurityContext`](#securitycontext)|SecurityContext defines the security options the container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext. More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/|
|`source`|`string`|Source contains the source code of the script to execute|
|`startupProbe`|[`Probe`](#probe)|StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully. If this probe fails, the Pod will be restarted, just as if the livenessProbe failed. This can be used to provide different probe parameters at the beginning of a Pod's lifecycle, when it might take a long time to load data or warm a cache, than during steady-state operation. This cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`stdin`|`boolean`|Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false.|
|`stdinOnce`|`boolean`|Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false|
|`terminationMessagePath`|`string`|Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated.|
|`terminationMessagePolicy`|`string`|Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated.|
|`tty`|`boolean`|Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false.|
|`volumeDevices`|`Array<`[`VolumeDevice`](#volumedevice)`>`|volumeDevices is the list of block devices to be used by the container.|
|`volumeMounts`|`Array<`[`VolumeMount`](#volumemount)`>`|Pod volumes to mount into the container's filesystem. Cannot be updated.|
|`workingDir`|`string`|Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.|

## WorkflowStep

WorkflowStep is a reference to a template to execute in a series of step

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`artifact-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`coinflip-recursive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip.yaml)

- [`conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-artifacts.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`conditionals-complex.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals-complex.yaml)

- [`conditionals.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals.yaml)

- [`continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/continue-on-fail.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-coinflip.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`exit-code-output-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-code-output-variable.yaml)

- [`exit-handler-step-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-step-level.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`exit-handlers.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handlers.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`fun-with-gifs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fun-with-gifs.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)

- [`hello-hybrid.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-hybrid.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`http-success-condition.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-success-condition.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`life-cycle-hooks-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-tmpl-level.yaml)

- [`life-cycle-hooks-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-wf-level.yaml)

- [`loops-maps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-sequence.yaml)

- [`loops.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-limit.yaml)

- [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-gc-strategy-with-label-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy-with-label-selector.yaml)

- [`pod-gc-strategy.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy.yaml)

- [`pod-metadata-wf-field.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata-wf-field.yaml)

- [`pod-metadata.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata.yaml)

- [`recursive-for-loop.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/recursive-for-loop.yaml)

- [`resubmit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/resubmit.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-python.yaml)

- [`status-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/status-reference.yaml)

- [`step-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/step-level-timeout.yaml)

- [`steps-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps-inline-workflow.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)

- [`suspend-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template.yaml)

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`template-defaults.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-defaults.yaml)

- [`template-on-exit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-on-exit.yaml)

- [`timeouts-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-workflow.yaml)

- [`volumes-existing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-pvc.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)

- [`event-consumer-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-event-binding/event-consumer-workflowtemplate.yaml)

- [`workflow-of-workflows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-of-workflows.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/steps.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`arguments`|[`Arguments`](#arguments)|Arguments hold arguments to the template|
|`continueOn`|[`ContinueOn`](#continueon)|ContinueOn makes argo to proceed with the following step even if this step fails. Errors and Failed states can be specified|
|`hooks`|[`LifecycleHook`](#lifecyclehook)|Hooks holds the lifecycle hook which is invoked at lifecycle of step, irrespective of the success, failure, or error status of the primary step|
|`inline`|[`Template`](#template)|Inline is the template. Template must be empty if this is declared (and vice-versa).|
|`name`|`string`|Name of the step|
|~`onExit`~|~`string`~|~OnExit is a template reference which is invoked at the end of the template, irrespective of the success, failure, or error of the primary template.~ DEPRECATED: Use Hooks[exit].Template instead.|
|`template`|`string`|Template is the name of the template to execute as the step|
|`templateRef`|[`TemplateRef`](#templateref)|TemplateRef is the reference to the template resource to execute as the step.|
|`when`|`string`|When is an expression in which the step should conditionally execute|
|`withItems`|`Array<`[`Item`](#item)`>`|WithItems expands a step into multiple parallel steps from the items in the list|
|`withParam`|`string`|WithParam expands a step into multiple parallel steps from the value in the parameter, which is expected to be a JSON list.|
|`withSequence`|[`Sequence`](#sequence)|WithSequence expands a step into a numeric sequence|

## SuspendTemplate

SuspendTemplate is a template subtype to suspend a workflow at a predetermined point in time

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`cron-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-workflow.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)

- [`suspend-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`duration`|`string`|Duration is the seconds to wait before automatically resuming a template|

## LabelValueFrom

_No description available_

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`label-value-from-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/label-value-from-workflow.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`expression`|`string`|_No description available_|

## ArtifactRepository

ArtifactRepository represents an artifact repository in which a controller will store its artifacts

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`archiveLogs`|`boolean`|ArchiveLogs enables log archiving|
|`artifactory`|[`ArtifactoryArtifactRepository`](#artifactoryartifactrepository)|Artifactory stores artifacts to JFrog Artifactory|
|`gcs`|[`GCSArtifactRepository`](#gcsartifactrepository)|GCS stores artifact in a GCS object store|
|`hdfs`|[`HDFSArtifactRepository`](#hdfsartifactrepository)|HDFS stores artifacts in HDFS|
|`oss`|[`OSSArtifactRepository`](#ossartifactrepository)|OSS stores artifact in a OSS-compliant object store|
|`s3`|[`S3ArtifactRepository`](#s3artifactrepository)|S3 stores artifact in a S3-compliant object store|

## MemoizationStatus

MemoizationStatus is the status of this memoized node

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`cacheName`|`string`|Cache is the name of the cache that was used|
|`hit`|`boolean`|Hit indicates whether this node was created from a cache entry|
|`key`|`string`|Key is the name of the key used for this node's cache|

## NodeSynchronizationStatus

NodeSynchronizationStatus stores the status of a node

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`waiting`|`string`|Waiting is the name of the lock that this node is waiting for|

## MutexStatus

MutexStatus contains which objects hold  mutex locks, and which objects this workflow is waiting on to release locks.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`synchronization-mutex-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-wf-level.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`holding`|`Array<`[`MutexHolding`](#mutexholding)`>`|Holding is a list of mutexes and their respective objects that are held by mutex lock for this io.argoproj.workflow.v1alpha1.|
|`waiting`|`Array<`[`MutexHolding`](#mutexholding)`>`|Waiting is a list of mutexes and their respective objects this workflow is waiting for.|

## SemaphoreStatus

_No description available_

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`holding`|`Array<`[`SemaphoreHolding`](#semaphoreholding)`>`|Holding stores the list of resource acquired synchronization lock for workflows.|
|`waiting`|`Array<`[`SemaphoreHolding`](#semaphoreholding)`>`|Waiting indicates the list of current synchronization lock holders.|

## ArchiveStrategy

ArchiveStrategy describes how to archive files/directory when saving artifacts

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`output-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-s3.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`none`|[`NoneStrategy`](#nonestrategy)|_No description available_|
|`tar`|[`TarStrategy`](#tarstrategy)|_No description available_|
|`zip`|[`ZipStrategy`](#zipstrategy)|_No description available_|

## ArtifactoryArtifact

ArtifactoryArtifact is the location of an artifactory artifact

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`passwordSecret`|[`SecretKeySelector`](#secretkeyselector)|PasswordSecret is the secret selector to the repository password|
|`url`|`string`|URL of the artifact|
|`usernameSecret`|[`SecretKeySelector`](#secretkeyselector)|UsernameSecret is the secret selector to the repository username|

## GCSArtifact

GCSArtifact is the location of a GCS artifact

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`input-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-gcs.yaml)

- [`output-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-gcs.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`bucket`|`string`|Bucket is the name of the bucket|
|`key`|`string`|Key is the path in the bucket where the artifact resides|
|`serviceAccountKeySecret`|[`SecretKeySelector`](#secretkeyselector)|ServiceAccountKeySecret is the secret selector to the bucket's service account key|

## GitArtifact

GitArtifact is the location of an git artifact

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`input-artifact-git.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-git.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`depth`|`integer`|Depth specifies clones/fetches should be shallow and include the given number of commits from the branch tip|
|`disableSubmodules`|`boolean`|DisableSubmodules disables submodules during git clone|
|`fetch`|`Array< string >`|Fetch specifies a number of refs that should be fetched before checkout|
|`insecureIgnoreHostKey`|`boolean`|InsecureIgnoreHostKey disables SSH strict host key checking during git clone|
|`passwordSecret`|[`SecretKeySelector`](#secretkeyselector)|PasswordSecret is the secret selector to the repository password|
|`repo`|`string`|Repo is the git repository|
|`revision`|`string`|Revision is the git commit, tag, branch to checkout|
|`sshPrivateKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SSHPrivateKeySecret is the secret selector to the repository ssh private key|
|`usernameSecret`|[`SecretKeySelector`](#secretkeyselector)|UsernameSecret is the secret selector to the repository username|

## HDFSArtifact

HDFSArtifact is the location of an HDFS artifact

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`addresses`|`Array< string >`|Addresses is accessible addresses of HDFS name nodes|
|`force`|`boolean`|Force copies a file forcibly even if it exists|
|`hdfsUser`|`string`|HDFSUser is the user to access HDFS file system. It is ignored if either ccache or keytab is used.|
|`krbCCacheSecret`|[`SecretKeySelector`](#secretkeyselector)|KrbCCacheSecret is the secret selector for Kerberos ccache Either ccache or keytab can be set to use Kerberos.|
|`krbConfigConfigMap`|[`ConfigMapKeySelector`](#configmapkeyselector)|KrbConfig is the configmap selector for Kerberos config as string It must be set if either ccache or keytab is used.|
|`krbKeytabSecret`|[`SecretKeySelector`](#secretkeyselector)|KrbKeytabSecret is the secret selector for Kerberos keytab Either ccache or keytab can be set to use Kerberos.|
|`krbRealm`|`string`|KrbRealm is the Kerberos realm used with Kerberos keytab It must be set if keytab is used.|
|`krbServicePrincipalName`|`string`|KrbServicePrincipalName is the principal name of Kerberos service It must be set if either ccache or keytab is used.|
|`krbUsername`|`string`|KrbUsername is the Kerberos username used with Kerberos keytab It must be set if keytab is used.|
|`path`|`string`|Path is a file path in HDFS|

## HTTPArtifact

HTTPArtifact allows a file served on HTTP to be placed as an input artifact in a container

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-artifacts.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`http-success-condition.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-success-condition.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`input-artifact-http.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-oss.yaml)

- [`life-cycle-hooks-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-tmpl-level.yaml)

- [`life-cycle-hooks-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-wf-level.yaml)

- [`sidecar-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`headers`|`Array<`[`Header`](#header)`>`|Headers are an optional list of headers to send with HTTP requests for artifacts|
|`passwordSecret`|[`SecretKeySelector`](#secretkeyselector)|PasswordSecret is the secret selector to the repository password|
|`url`|`string`|URL of the artifact|
|`usernameSecret`|[`SecretKeySelector`](#secretkeyselector)|UsernameSecret is the secret selector to the repository username|

## OSSArtifact

OSSArtifact is the location of an Alibaba Cloud OSS artifact

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`input-artifact-oss.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-oss.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`accessKeySecret`|[`SecretKeySelector`](#secretkeyselector)|AccessKeySecret is the secret selector to the bucket's access key|
|`bucket`|`string`|Bucket is the name of the bucket|
|`createBucketIfNotPresent`|`boolean`|CreateBucketIfNotPresent tells the driver to attempt to create the OSS bucket for output artifacts, if it doesn't exist|
|`endpoint`|`string`|Endpoint is the hostname of the bucket endpoint|
|`key`|`string`|Key is the path in the bucket where the artifact resides|
|`lifecycleRule`|[`OSSLifecycleRule`](#osslifecyclerule)|LifecycleRule specifies how to manage bucket's lifecycle|
|`secretKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySecret is the secret selector to the bucket's secret key|
|`securityToken`|`string`|SecurityToken is the user's temporary security token. For more details, check out: https://www.alibabacloud.com/help/doc-detail/100624.htm|

## RawArtifact

RawArtifact allows raw string content to be placed as an artifact in a container

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`input-artifact-raw.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-raw.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`data`|`string`|Data is the string contents of the artifact|

## S3Artifact

S3Artifact is the location of an S3 artifact

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`accessKeySecret`|[`SecretKeySelector`](#secretkeyselector)|AccessKeySecret is the secret selector to the bucket's access key|
|`bucket`|`string`|Bucket is the name of the bucket|
|`createBucketIfNotPresent`|[`CreateS3BucketOptions`](#creates3bucketoptions)|CreateBucketIfNotPresent tells the driver to attempt to create the S3 bucket for output artifacts, if it doesn't exist. Setting Enabled Encryption will apply either SSE-S3 to the bucket if KmsKeyId is not set or SSE-KMS if it is.|
|`encryptionOptions`|[`S3EncryptionOptions`](#s3encryptionoptions)|_No description available_|
|`endpoint`|`string`|Endpoint is the hostname of the bucket endpoint|
|`insecure`|`boolean`|Insecure will connect to the service with TLS|
|`key`|`string`|Key is the key in the bucket where the artifact resides|
|`region`|`string`|Region contains the optional bucket region|
|`roleARN`|`string`|RoleARN is the Amazon Resource Name (ARN) of the role to assume.|
|`secretKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySecret is the secret selector to the bucket's secret key|
|`useSDKCreds`|`boolean`|UseSDKCreds tells the driver to figure out credentials based on sdk defaults.|

## ValueFrom

ValueFrom describes a location in which to obtain the value to a parameter

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`configMapKeyRef`|[`ConfigMapKeySelector`](#configmapkeyselector)|ConfigMapKeyRef is configmap selector for input parameter configuration|
|`default`|`string`|Default specifies a value to be used if retrieving the value from the specified source fails|
|`event`|`string`|Selector (https://github.com/antonmedv/expr) that is evaluated against the event to get the value of the parameter. E.g. `payload.message`|
|`expression`|`string`|Expression, if defined, is evaluated to specify the value for the parameter|
|`jqFilter`|`string`|JQFilter expression against the resource object in resource templates|
|`jsonPath`|`string`|JSONPath of a resource to retrieve an output parameter value from in resource templates|
|`parameter`|`string`|Parameter reference to a step or dag task in which to retrieve an output parameter value from (e.g. '{{steps.mystep.outputs.myparam}}')|
|`path`|`string`|Path in the container to retrieve an output parameter value from in container templates|
|`supplied`|[`SuppliedValueFrom`](#suppliedvaluefrom)|Supplied value to be filled in directly, either through the CLI, API, etc.|

## Counter

Counter is a Counter prometheus metric

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`value`|`string`|Value is the value of the metric|

## Gauge

Gauge is a Gauge prometheus metric

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`realtime`|`boolean`|Realtime emits this metric in real time if applicable|
|`value`|`string`|Value is the value of the metric|

## Histogram

Histogram is a Histogram prometheus metric

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`buckets`|`Array<`[`Amount`](#amount)`>`|Buckets is a list of bucket divisors for the histogram|
|`value`|`string`|Value is the value of the metric|

## MetricLabel

MetricLabel is a single label for a prometheus metric

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-artifacts.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`graph-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/graph-workflow.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`parallel-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/parallel-workflow.yaml)

- [`sequence-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/sequence-workflow.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`dag-conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-artifacts.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflow.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-world.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`http-success-condition.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-success-condition.yaml)

- [`k8s-owner-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-owner-reference.yaml)

- [`k8s-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-patch.yaml)

- [`pod-gc-strategy-with-label-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy-with-label-selector.yaml)

- [`pod-metadata-wf-field.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata-wf-field.yaml)

- [`pod-metadata.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata.yaml)

- [`selected-executor-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/selected-executor-workflow.yaml)

- [`steps-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps-inline-workflow.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|_No description available_|
|`value`|`string`|_No description available_|

## RetryNodeAntiAffinity

RetryNodeAntiAffinity is a placeholder for future expansion, only empty nodeAntiAffinity is allowed. In order to prevent running steps on the same host, it uses "kubernetes.io/hostname".

## ContainerNode

_No description available_

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`graph-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/graph-workflow.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`parallel-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/parallel-workflow.yaml)

- [`sequence-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/sequence-workflow.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-yaml-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-yaml-patch.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`args`|`Array< string >`|Arguments to the entrypoint. The docker image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell|
|`command`|`Array< string >`|Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell|
|`dependencies`|`Array< string >`|_No description available_|
|`env`|`Array<`[`EnvVar`](#envvar)`>`|List of environment variables to set in the container. Cannot be updated.|
|`envFrom`|`Array<`[`EnvFromSource`](#envfromsource)`>`|List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.|
|`image`|`string`|Docker image name. More info: https://kubernetes.io/docs/concepts/containers/images This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets.|
|`imagePullPolicy`|`string`|Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images|
|`lifecycle`|[`Lifecycle`](#lifecycle)|Actions that the management system should take in response to container lifecycle events. Cannot be updated.|
|`livenessProbe`|[`Probe`](#probe)|Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`name`|`string`|Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.|
|`ports`|`Array<`[`ContainerPort`](#containerport)`>`|List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated.|
|`readinessProbe`|[`Probe`](#probe)|Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`resources`|[`ResourceRequirements`](#resourcerequirements)|Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/|
|`securityContext`|[`SecurityContext`](#securitycontext)|SecurityContext defines the security options the container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext. More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/|
|`startupProbe`|[`Probe`](#probe)|StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully. If this probe fails, the Pod will be restarted, just as if the livenessProbe failed. This can be used to provide different probe parameters at the beginning of a Pod's lifecycle, when it might take a long time to load data or warm a cache, than during steady-state operation. This cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`stdin`|`boolean`|Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false.|
|`stdinOnce`|`boolean`|Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false|
|`terminationMessagePath`|`string`|Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated.|
|`terminationMessagePolicy`|`string`|Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated.|
|`tty`|`boolean`|Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false.|
|`volumeDevices`|`Array<`[`VolumeDevice`](#volumedevice)`>`|volumeDevices is the list of block devices to be used by the container.|
|`volumeMounts`|`Array<`[`VolumeMount`](#volumemount)`>`|Pod volumes to mount into the container's filesystem. Cannot be updated.|
|`workingDir`|`string`|Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.|

## ContainerSetRetryStrategy

_No description available_

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`dag-disable-failFast.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-disable-failFast.yaml)

- [`retry-backoff.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-backoff.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`retry-container-to-completion.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container.yaml)

- [`retry-on-error.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-on-error.yaml)

- [`retry-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-script.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-with-steps.yaml)

- [`template-defaults.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-defaults.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`duration`|`string`|Duration is the time between each retry, examples values are "300ms", "1s" or "5m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".|
|`retries`|[`IntOrString`](#intorstring)|Nbr of retries|

## DAGTask

DAGTask represents a node in the graph during DAG execution

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`cluster-wftmpl-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`dag-coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-coinflip.yaml)

- [`dag-conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-artifacts.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`dag-continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-continue-on-fail.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-disable-failFast.yaml)

- [`dag-enhanced-depends.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-enhanced-depends.yaml)

- [`dag-inline-clusterworkflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-clusterworkflowtemplate.yaml)

- [`dag-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflow.yaml)

- [`dag-inline-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflowtemplate.yaml)

- [`dag-multiroot.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-nested.yaml)

- [`dag-targets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-targets.yaml)

- [`dag-task-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-task-level-timeout.yaml)

- [`exit-handler-dag-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-dag-level.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`key-only-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/key-only-artifact.yaml)

- [`loops-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-dag.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-dag.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`resubmit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/resubmit.yaml)

- [`dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/dag.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`arguments`|[`Arguments`](#arguments)|Arguments are the parameter and artifact arguments to the template|
|`continueOn`|[`ContinueOn`](#continueon)|ContinueOn makes argo to proceed with the following step even if this step fails. Errors and Failed states can be specified|
|`dependencies`|`Array< string >`|Dependencies are name of other targets which this depends on|
|`depends`|`string`|Depends are name of other targets which this depends on|
|`hooks`|[`LifecycleHook`](#lifecyclehook)|Hooks hold the lifecycle hook which is invoked at lifecycle of task, irrespective of the success, failure, or error status of the primary task|
|`inline`|[`Template`](#template)|Inline is the template. Template must be empty if this is declared (and vice-versa).|
|`name`|`string`|Name is the name of the target|
|~`onExit`~|~`string`~|~OnExit is a template reference which is invoked at the end of the template, irrespective of the success, failure, or error of the primary template.~ DEPRECATED: Use Hooks[exit].Template instead.|
|`template`|`string`|Name of template to execute|
|`templateRef`|[`TemplateRef`](#templateref)|TemplateRef is the reference to the template resource to execute.|
|`when`|`string`|When is an expression in which the task should conditionally execute|
|`withItems`|`Array<`[`Item`](#item)`>`|WithItems expands a task into multiple parallel tasks from the items in the list|
|`withParam`|`string`|WithParam expands a task into multiple parallel tasks from the value in the parameter, which is expected to be a JSON list.|
|`withSequence`|[`Sequence`](#sequence)|WithSequence expands a task into a numeric sequence|

## DataSource

DataSource sources external data into a data template

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`coinflip-recursive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip.yaml)

- [`colored-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/colored-logs.yaml)

- [`conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-artifacts.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`conditionals-complex.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals-complex.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`dag-coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-coinflip.yaml)

- [`dag-conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-artifacts.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`expression-destructure-json.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-destructure-json.yaml)

- [`expression-reusing-verbose-snippets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-reusing-verbose-snippets.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`loops-param-result.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-result.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`parameter-aggregation-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`recursive-for-loop.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/recursive-for-loop.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`retry-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-script.yaml)

- [`scripts-bash.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-python.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifactPaths`|[`ArtifactPaths`](#artifactpaths)|ArtifactPaths is a data transformation that collects a list of artifact paths|

## TransformationStep

_No description available_

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`expression`|`string`|Expression defines an expr expression to apply|

## HTTPHeader

_No description available_

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|_No description available_|
|`value`|`string`|_No description available_|
|`valueFrom`|[`HTTPHeaderSource`](#httpheadersource)|_No description available_|

## Cache

Cache is the configuration for the type of cache to be used

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`configMap`|[`ConfigMapKeySelector`](#configmapkeyselector)|ConfigMap sets a ConfigMap-based cache|

## ContinueOn

ContinueOn defines if a workflow should continue even if a task or step fails/errors. It can be specified if the workflow should continue when the pod errors, fails or both.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/continue-on-fail.yaml)

- [`exit-code-output-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-code-output-variable.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`status-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/status-reference.yaml)

- [`step-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/step-level-timeout.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`error`|`boolean`|_No description available_|
|`failed`|`boolean`|_No description available_|

## Item

Item expands a single workflow step into multiple parallel steps The value of Item can be a map, string, bool, or number

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`loops-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-dag.yaml)

- [`loops-maps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-maps.yaml)

- [`loops.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops.yaml)

- [`parallelism-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-limit.yaml)

- [`parallelism-template-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`timeouts-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-workflow.yaml)
</details>

## Sequence

Sequence expands a workflow step into numeric range

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`loops-sequence.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-sequence.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`count`|[`IntOrString`](#intorstring)|Count is number of elements in the sequence (default: 0). Not to be used with end|
|`end`|[`IntOrString`](#intorstring)|Number at which to end the sequence (default: 0). Not to be used with Count|
|`format`|`string`|Format is a printf format string to format the value in the sequence|
|`start`|[`IntOrString`](#intorstring)|Number at which to start the sequence (default: 0)|

## ArtifactoryArtifactRepository

ArtifactoryArtifactRepository defines the controller configuration for an artifactory artifact repository

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`passwordSecret`|[`SecretKeySelector`](#secretkeyselector)|PasswordSecret is the secret selector to the repository password|
|`repoURL`|`string`|RepoURL is the url for artifactory repo.|
|`usernameSecret`|[`SecretKeySelector`](#secretkeyselector)|UsernameSecret is the secret selector to the repository username|

## GCSArtifactRepository

GCSArtifactRepository defines the controller configuration for a GCS artifact repository

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`input-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-gcs.yaml)

- [`output-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-gcs.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`bucket`|`string`|Bucket is the name of the bucket|
|`keyFormat`|`string`|KeyFormat is defines the format of how to store keys. Can reference workflow variables|
|`serviceAccountKeySecret`|[`SecretKeySelector`](#secretkeyselector)|ServiceAccountKeySecret is the secret selector to the bucket's service account key|

## HDFSArtifactRepository

HDFSArtifactRepository defines the controller configuration for an HDFS artifact repository

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`addresses`|`Array< string >`|Addresses is accessible addresses of HDFS name nodes|
|`force`|`boolean`|Force copies a file forcibly even if it exists|
|`hdfsUser`|`string`|HDFSUser is the user to access HDFS file system. It is ignored if either ccache or keytab is used.|
|`krbCCacheSecret`|[`SecretKeySelector`](#secretkeyselector)|KrbCCacheSecret is the secret selector for Kerberos ccache Either ccache or keytab can be set to use Kerberos.|
|`krbConfigConfigMap`|[`ConfigMapKeySelector`](#configmapkeyselector)|KrbConfig is the configmap selector for Kerberos config as string It must be set if either ccache or keytab is used.|
|`krbKeytabSecret`|[`SecretKeySelector`](#secretkeyselector)|KrbKeytabSecret is the secret selector for Kerberos keytab Either ccache or keytab can be set to use Kerberos.|
|`krbRealm`|`string`|KrbRealm is the Kerberos realm used with Kerberos keytab It must be set if keytab is used.|
|`krbServicePrincipalName`|`string`|KrbServicePrincipalName is the principal name of Kerberos service It must be set if either ccache or keytab is used.|
|`krbUsername`|`string`|KrbUsername is the Kerberos username used with Kerberos keytab It must be set if keytab is used.|
|`pathFormat`|`string`|PathFormat is defines the format of path to store a file. Can reference workflow variables|

## OSSArtifactRepository

OSSArtifactRepository defines the controller configuration for an OSS artifact repository

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`input-artifact-oss.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-oss.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`accessKeySecret`|[`SecretKeySelector`](#secretkeyselector)|AccessKeySecret is the secret selector to the bucket's access key|
|`bucket`|`string`|Bucket is the name of the bucket|
|`createBucketIfNotPresent`|`boolean`|CreateBucketIfNotPresent tells the driver to attempt to create the OSS bucket for output artifacts, if it doesn't exist|
|`endpoint`|`string`|Endpoint is the hostname of the bucket endpoint|
|`keyFormat`|`string`|KeyFormat is defines the format of how to store keys. Can reference workflow variables|
|`lifecycleRule`|[`OSSLifecycleRule`](#osslifecyclerule)|LifecycleRule specifies how to manage bucket's lifecycle|
|`secretKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySecret is the secret selector to the bucket's secret key|
|`securityToken`|`string`|SecurityToken is the user's temporary security token. For more details, check out: https://www.alibabacloud.com/help/doc-detail/100624.htm|

## S3ArtifactRepository

S3ArtifactRepository defines the controller configuration for an S3 artifact repository

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`accessKeySecret`|[`SecretKeySelector`](#secretkeyselector)|AccessKeySecret is the secret selector to the bucket's access key|
|`bucket`|`string`|Bucket is the name of the bucket|
|`createBucketIfNotPresent`|[`CreateS3BucketOptions`](#creates3bucketoptions)|CreateBucketIfNotPresent tells the driver to attempt to create the S3 bucket for output artifacts, if it doesn't exist. Setting Enabled Encryption will apply either SSE-S3 to the bucket if KmsKeyId is not set or SSE-KMS if it is.|
|`encryptionOptions`|[`S3EncryptionOptions`](#s3encryptionoptions)|_No description available_|
|`endpoint`|`string`|Endpoint is the hostname of the bucket endpoint|
|`insecure`|`boolean`|Insecure will connect to the service with TLS|
|`keyFormat`|`string`|KeyFormat is defines the format of how to store keys. Can reference workflow variables|
|~`keyPrefix`~|~`string`~|~KeyPrefix is prefix used as part of the bucket key in which the controller will store artifacts.~ DEPRECATED. Use KeyFormat instead|
|`region`|`string`|Region contains the optional bucket region|
|`roleARN`|`string`|RoleARN is the Amazon Resource Name (ARN) of the role to assume.|
|`secretKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySecret is the secret selector to the bucket's secret key|
|`useSDKCreds`|`boolean`|UseSDKCreds tells the driver to figure out credentials based on sdk defaults.|

## MutexHolding

MutexHolding describes the mutex and the object which is holding it.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`holder`|`string`|Holder is a reference to the object which holds the Mutex. Holding Scenario:  1. Current workflow's NodeID which is holding the lock.     e.g: ${NodeID}Waiting Scenario:  1. Current workflow or other workflow NodeID which is holding the lock.     e.g: ${WorkflowName}/${NodeID}|
|`mutex`|`string`|Reference for the mutex e.g: ${namespace}/mutex/${mutexName}|

## SemaphoreHolding

_No description available_

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`holders`|`Array< string >`|Holders stores the list of current holder names in the io.argoproj.workflow.v1alpha1.|
|`semaphore`|`string`|Semaphore stores the semaphore name.|

## NoneStrategy

NoneStrategy indicates to skip tar process and upload the files or directory tree as independent files. Note that if the artifact is a directory, the artifact driver must support the ability to save/load the directory appropriately.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`output-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-s3.yaml)
</details>

## TarStrategy

TarStrategy will tar and gzip the file or directory when saving

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`compressionLevel`|`integer`|CompressionLevel specifies the gzip compression level to use for the artifact. Defaults to gzip.DefaultCompression.|

## ZipStrategy

ZipStrategy will unzip zipped input artifacts

## Header

Header indicate a key-value request header to be used when fetching artifacts over HTTP

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|Name is the header name|
|`value`|`string`|Value is the literal value to use for the header|

## OSSLifecycleRule

OSSLifecycleRule specifies how to manage bucket's lifecycle

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`markDeletionAfterDays`|`integer`|MarkDeletionAfterDays is the number of days before we delete objects in the bucket|
|`markInfrequentAccessAfterDays`|`integer`|MarkInfrequentAccessAfterDays is the number of days before we convert the objects in the bucket to Infrequent Access (IA) storage type|

## CreateS3BucketOptions

CreateS3BucketOptions options used to determine automatic automatic bucket-creation process

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`objectLocking`|`boolean`|ObjectLocking Enable object locking|

## S3EncryptionOptions

S3EncryptionOptions used to determine encryption options during s3 operations

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`enableEncryption`|`boolean`|EnableEncryption tells the driver to encrypt objects if set to true. If kmsKeyId and serverSideCustomerKeySecret are not set, SSE-S3 will be used|
|`kmsEncryptionContext`|`string`|KmsEncryptionContext is a json blob that contains an encryption context. See https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html#encrypt_context for more information|
|`kmsKeyId`|`string`|KMSKeyId tells the driver to encrypt the object using the specified KMS Key.|
|`serverSideCustomerKeySecret`|[`SecretKeySelector`](#secretkeyselector)|ServerSideCustomerKeySecret tells the driver to encrypt the output artifacts using SSE-C with the specified secret.|

## SuppliedValueFrom

SuppliedValueFrom is a placeholder for a value to be filled in directly, either through the CLI, API, etc.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)
</details>

## Amount

Amount represent a numeric amount.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)
</details>

## ArtifactPaths

ArtifactPaths expands a step from a collection of artifacts

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`archive`|[`ArchiveStrategy`](#archivestrategy)|Archive controls how the artifact will be saved to the artifact repository.|
|`archiveLogs`|`boolean`|ArchiveLogs indicates if the container logs should be archived|
|`artifactory`|[`ArtifactoryArtifact`](#artifactoryartifact)|Artifactory contains artifactory artifact location details|
|`from`|`string`|From allows an artifact to reference an artifact from a previous step|
|`fromExpression`|`string`|FromExpression, if defined, is evaluated to specify the value for the artifact|
|`gcs`|[`GCSArtifact`](#gcsartifact)|GCS contains GCS artifact location details|
|`git`|[`GitArtifact`](#gitartifact)|Git contains git artifact location details|
|`globalName`|`string`|GlobalName exports an output artifact to the global scope, making it available as '{{io.argoproj.workflow.v1alpha1.outputs.artifacts.XXXX}} and in workflow.status.outputs.artifacts|
|`hdfs`|[`HDFSArtifact`](#hdfsartifact)|HDFS contains HDFS artifact location details|
|`http`|[`HTTPArtifact`](#httpartifact)|HTTP contains HTTP artifact location details|
|`mode`|`integer`|mode bits to use on this file, must be a value between 0 and 0777 set when loading input artifacts.|
|`name`|`string`|name of the artifact. must be unique within a template's inputs/outputs.|
|`optional`|`boolean`|Make Artifacts optional, if Artifacts doesn't generate or exist|
|`oss`|[`OSSArtifact`](#ossartifact)|OSS contains OSS artifact location details|
|`path`|`string`|Path is the container path to the artifact|
|`raw`|[`RawArtifact`](#rawartifact)|Raw contains raw artifact location details|
|`recurseMode`|`boolean`|If mode is set, apply the permission recursively into the artifact if it is a folder|
|`s3`|[`S3Artifact`](#s3artifact)|S3 contains S3 artifact location details|
|`subPath`|`string`|SubPath allows an artifact to be sourced from a subpath within the specified source|

## HTTPHeaderSource

_No description available_

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`secretKeyRef`|[`SecretKeySelector`](#secretkeyselector)|_No description available_|

# External Fields


## ObjectMeta

ObjectMeta is metadata that all persisted resources must have, which includes all objects users must create.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-artifacts.yaml)

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`arguments-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`artifact-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`artifact-repository-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-repository-ref.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`workflow-template-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref.yaml)

- [`coinflip-recursive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip.yaml)

- [`colored-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/colored-logs.yaml)

- [`conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-artifacts.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`conditionals-complex.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals-complex.yaml)

- [`conditionals.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals.yaml)

- [`graph-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/graph-workflow.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`parallel-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/parallel-workflow.yaml)

- [`sequence-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/sequence-workflow.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/continue-on-fail.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`cron-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-coinflip.yaml)

- [`dag-conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-artifacts.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`dag-continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-continue-on-fail.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-disable-failFast.yaml)

- [`dag-enhanced-depends.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-enhanced-depends.yaml)

- [`dag-inline-clusterworkflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-clusterworkflowtemplate.yaml)

- [`dag-inline-cronworkflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-cronworkflow.yaml)

- [`dag-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflow.yaml)

- [`dag-inline-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflowtemplate.yaml)

- [`dag-multiroot.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-nested.yaml)

- [`dag-targets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-targets.yaml)

- [`dag-task-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-task-level-timeout.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`default-pdb-support.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/default-pdb-support.yaml)

- [`dns-config.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dns-config.yaml)

- [`exit-code-output-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-code-output-variable.yaml)

- [`exit-handler-dag-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-dag-level.yaml)

- [`exit-handler-slack.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-slack.yaml)

- [`exit-handler-step-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-step-level.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`exit-handlers.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handlers.yaml)

- [`expression-destructure-json.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-destructure-json.yaml)

- [`expression-reusing-verbose-snippets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-reusing-verbose-snippets.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`forever.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/forever.yaml)

- [`fun-with-gifs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/gc-ttl.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)

- [`global-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)

- [`hello-hybrid.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-hybrid.yaml)

- [`hello-windows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-windows.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-world.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`http-success-condition.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-success-condition.yaml)

- [`image-pull-secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`init-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-s3.yaml)

- [`k8s-json-patch-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-json-patch-workflow.yaml)

- [`k8s-owner-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-owner-reference.yaml)

- [`k8s-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-patch.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`key-only-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/key-only-artifact.yaml)

- [`label-value-from-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/label-value-from-workflow.yaml)

- [`life-cycle-hooks-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-tmpl-level.yaml)

- [`life-cycle-hooks-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-wf-level.yaml)

- [`loops-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-dag.yaml)

- [`loops-maps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-sequence.yaml)

- [`loops.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`node-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/node-selector.yaml)

- [`output-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-gcs.yaml)

- [`output-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-gc-strategy-with-label-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy-with-label-selector.yaml)

- [`pod-gc-strategy.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy.yaml)

- [`pod-metadata-wf-field.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata-wf-field.yaml)

- [`pod-metadata.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-yaml-patch.yaml)

- [`recursive-for-loop.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/recursive-for-loop.yaml)

- [`resubmit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/resubmit.yaml)

- [`retry-backoff.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-backoff.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`retry-container-to-completion.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container.yaml)

- [`retry-on-error.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-on-error.yaml)

- [`retry-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-script.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-python.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)

- [`selected-executor-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/selected-executor-workflow.yaml)

- [`sidecar-dind.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar.yaml)

- [`status-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/status-reference.yaml)

- [`step-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/step-level-timeout.yaml)

- [`steps-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps-inline-workflow.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)

- [`suspend-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template.yaml)

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`synchronization-mutex-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-wf-level.yaml)

- [`template-defaults.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-defaults.yaml)

- [`template-on-exit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-on-exit.yaml)

- [`timeouts-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-pvc.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)

- [`event-consumer-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-event-binding/event-consumer-workflowtemplate.yaml)

- [`workflow-of-workflows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-of-workflows.yaml)

- [`dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/steps.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)

- [`workflow-archive-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-archive-logs.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`workflow-template-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`annotations`|`Map< string , string >`|Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. More info: http://kubernetes.io/docs/user-guide/annotations|
|`clusterName`|`string`|The name of the cluster which the object belongs to. This is used to distinguish resources with same name and namespace in different clusters. This field is not set anywhere right now and apiserver is going to ignore it if set in create or update request.|
|`creationTimestamp`|[`Time`](#time)|CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations. Clients may not set this value. It is represented in RFC3339 form and is in UTC.Populated by the system. Read-only. Null for lists. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata|
|`deletionGracePeriodSeconds`|`integer`|Number of seconds allowed for this object to gracefully terminate before it will be removed from the system. Only set when deletionTimestamp is also set. May only be shortened. Read-only.|
|`deletionTimestamp`|[`Time`](#time)|DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. This field is set by the server when a graceful deletion is requested by the user, and is not directly settable by a client. The resource is expected to be deleted (no longer visible from resource lists, and not reachable by name) after the time in this field, once the finalizers list is empty. As long as the finalizers list contains items, deletion is blocked. Once the deletionTimestamp is set, this value may not be unset or be set further into the future, although it may be shortened or the resource may be deleted prior to this time. For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will react by sending a graceful termination signal to the containers in the pod. After that 30 seconds, the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup, remove the pod from the API. In the presence of network partitions, this object may still exist after this timestamp, until an administrator or automated process can determine the resource is fully terminated. If not set, graceful deletion of the object has not been requested.Populated by the system when a graceful deletion is requested. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata|
|`finalizers`|`Array< string >`|Must be empty before the object is deleted from the registry. Each entry is an identifier for the responsible component that will remove the entry from the list. If the deletionTimestamp of the object is non-nil, entries in this list can only be removed. Finalizers may be processed and removed in any order.  Order is NOT enforced because it introduces significant risk of stuck finalizers. finalizers is a shared field, any actor with permission can reorder it. If the finalizer list is processed in order, then this can lead to a situation in which the component responsible for the first finalizer in the list is waiting for a signal (field value, external system, or other) produced by a component responsible for a finalizer later in the list, resulting in a deadlock. Without enforced ordering finalizers are free to order amongst themselves and are not vulnerable to ordering changes in the list.|
|`generateName`|`string`|GenerateName is an optional prefix, used by the server, to generate a unique name ONLY IF the Name field has not been provided. If this field is used, the name returned to the client will be different than the name passed. This value will also be combined with a unique suffix. The provided value has the same validation rules as the Name field, and may be truncated by the length of the suffix required to make the value unique on the server.If this field is specified and the generated name exists, the server will NOT return a 409 - instead, it will either return 201 Created or 500 with Reason ServerTimeout indicating a unique name could not be found in the time allotted, and the client should retry (optionally after the time indicated in the Retry-After header).Applied only if Name is not specified. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency|
|`generation`|`integer`|A sequence number representing a specific generation of the desired state. Populated by the system. Read-only.|
|`labels`|`Map< string , string >`|Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels|
|`managedFields`|`Array<`[`ManagedFieldsEntry`](#managedfieldsentry)`>`|ManagedFields maps workflow-id and version to the set of fields that are managed by that workflow. This is mostly for internal housekeeping, and users typically shouldn't need to set or understand this field. A workflow can be the user's name, a controller's name, or the name of a specific apply path like "ci-cd". The set of fields is always in the version that the workflow used when modifying the object.|
|`name`|`string`|Name must be unique within a namespace. Is required when creating resources, although some resources may allow a client to request the generation of an appropriate name automatically. Name is primarily intended for creation idempotence and configuration definition. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names|
|`namespace`|`string`|Namespace defines the space within which each name must be unique. An empty namespace is equivalent to the "default" namespace, but "default" is the canonical representation. Not all objects are required to be scoped to a namespace - the value of this field for those objects will be empty.Must be a DNS_LABEL. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/namespaces|
|`ownerReferences`|`Array<`[`OwnerReference`](#ownerreference)`>`|List of objects depended by this object. If ALL objects in the list have been deleted, this object will be garbage collected. If this object is managed by a controller, then an entry in this list will point to this controller, with the controller field set to true. There cannot be more than one managing controller.|
|`resourceVersion`|`string`|An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed. May be used for optimistic concurrency, change detection, and the watch operation on a resource or set of resources. Clients must treat these values as opaque and passed unmodified back to the server. They may only be valid for a particular resource or set of resources.Populated by the system. Read-only. Value must be treated as opaque by clients and . More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency|
|~`selfLink`~|~`string`~|~SelfLink is a URL representing this object. Populated by the system. Read-only~ DEPRECATED Kubernetes will stop propagating this field in 1.20 release and the field is planned to be removed in 1.21 release.|
|`uid`|`string`|UID is the unique in time and space value for this object. It is typically generated by the server on successful creation of a resource and is not allowed to change on PUT operations.Populated by the system. Read-only. More info: http://kubernetes.io/docs/user-guide/identifiers#uids|

## Affinity

Affinity is a group of affinity scheduling rules.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`nodeAffinity`|[`NodeAffinity`](#nodeaffinity)|Describes node affinity scheduling rules for the pod.|
|`podAffinity`|[`PodAffinity`](#podaffinity)|Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).|
|`podAntiAffinity`|[`PodAntiAffinity`](#podantiaffinity)|Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).|

## PodDNSConfig

PodDNSConfig defines the DNS parameters of a pod in addition to those generated from DNSPolicy.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`dns-config.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dns-config.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`nameservers`|`Array< string >`|A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed.|
|`options`|`Array<`[`PodDNSConfigOption`](#poddnsconfigoption)`>`|A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy.|
|`searches`|`Array< string >`|A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed.|

## HostAlias

HostAlias holds the mapping between IP and hostnames that will be injected as an entry in the pod's hosts file.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`hostnames`|`Array< string >`|Hostnames for the above IP address.|
|`ip`|`string`|IP address of the host file entry.|

## LocalObjectReference

LocalObjectReference contains enough information to let you locate the referenced object inside the same namespace.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`image-pull-secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/image-pull-secrets.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names|

## PodDisruptionBudgetSpec

PodDisruptionBudgetSpec is a description of a PodDisruptionBudget.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`default-pdb-support.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/default-pdb-support.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`maxUnavailable`|[`IntOrString`](#intorstring)|An eviction is allowed if at most "maxUnavailable" pods selected by "selector" are unavailable after the eviction, i.e. even in absence of the evicted pod. For example, one can prevent all voluntary evictions by specifying 0. This is a mutually exclusive setting with "minAvailable".|
|`minAvailable`|[`IntOrString`](#intorstring)|An eviction is allowed if at least "minAvailable" pods selected by "selector" will still be available after the eviction, i.e. even in the absence of the evicted pod.  So for example you can prevent all voluntary evictions by specifying "100%".|
|`selector`|[`LabelSelector`](#labelselector)|Label query over pods whose evictions are managed by the disruption budget. A null selector selects no pods. An empty selector ({}) also selects no pods, which differs from standard behavior of selecting all pods. In policy/v1, an empty selector will select all pods in the namespace.|

## PodSecurityContext

PodSecurityContext holds pod-level security attributes and common container settings. Some fields are also present in container.securityContext.  Field values of container.securityContext take precedence over field values of PodSecurityContext.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`sidecar-dind.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-dind.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsGroup`|`integer`|A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod:1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR'd with rw-rw----If unset, the Kubelet will not modify the ownership and permissions of any volume. Note that this field cannot be set when spec.os.name is windows.|
|`fsGroupChangePolicy`|`string`|fsGroupChangePolicy defines behavior of changing ownership and permission of the volume before being exposed inside Pod. This field will only apply to volume types which support fsGroup based ownership(and permissions). It will have no effect on ephemeral volume types such as: secret, configmaps and emptydir. Valid values are "OnRootMismatch" and "Always". If not specified, "Always" is used. Note that this field cannot be set when spec.os.name is windows.|
|`runAsGroup`|`integer`|The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. Note that this field cannot be set when spec.os.name is windows.|
|`runAsNonRoot`|`boolean`|Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.|
|`runAsUser`|`integer`|The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. Note that this field cannot be set when spec.os.name is windows.|
|`seLinuxOptions`|[`SELinuxOptions`](#selinuxoptions)|The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. Note that this field cannot be set when spec.os.name is windows.|
|`seccompProfile`|[`SeccompProfile`](#seccompprofile)|The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.|
|`supplementalGroups`|`Array< integer >`|A list of groups applied to the first process run in each container, in addition to the container's primary GID.  If unspecified, no groups will be added to any container. Note that this field cannot be set when spec.os.name is windows.|
|`sysctls`|`Array<`[`Sysctl`](#sysctl)`>`|Sysctls hold a list of namespaced sysctls used for the pod. Pods with unsupported sysctls (by the container runtime) might fail to launch. Note that this field cannot be set when spec.os.name is windows.|
|`windowsOptions`|[`WindowsSecurityContextOptions`](#windowssecuritycontextoptions)|The Windows specific settings applied to all containers. If unspecified, the options within a container's SecurityContext will be used. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is linux.|

## Toleration

The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`effect`|`string`|Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.Possible enum values: - `"NoExecute"` Evict any already-running pods that do not tolerate the taint. Currently enforced by NodeController. - `"NoSchedule"` Do not allow new pods to schedule onto the node unless they tolerate the taint, but allow all pods submitted to Kubelet without going through the scheduler to start, and allow all already-running pods to continue running. Enforced by the scheduler. - `"PreferNoSchedule"` Like TaintEffectNoSchedule, but the scheduler tries not to schedule new pods onto the node, rather than prohibiting new pods from scheduling onto the node entirely. Enforced by the scheduler.|
|`key`|`string`|Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.|
|`operator`|`string`|Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.Possible enum values: - `"Equal"` - `"Exists"`|
|`tolerationSeconds`|`integer`|TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.|
|`value`|`string`|Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.|

## PersistentVolumeClaim

PersistentVolumeClaim is a user's request for and claim to a persistent volume

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`fun-with-gifs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fun-with-gifs.yaml)

- [`volumes-pvc.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-pvc.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiVersion`|`string`|APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources|
|`kind`|`string`|Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds|
|`metadata`|[`ObjectMeta`](#objectmeta)|Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata|
|`spec`|[`PersistentVolumeClaimSpec`](#persistentvolumeclaimspec)|Spec defines the desired characteristics of a volume requested by a pod author. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims|
|`status`|[`PersistentVolumeClaimStatus`](#persistentvolumeclaimstatus)|Status represents the current information/status of a persistent volume claim. Read-only. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims|

## Volume

Volume represents a named volume in a pod that may be accessed by any container in the pod.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`init-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/init-container.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)

- [`volumes-emptydir.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-existing.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`awsElasticBlockStore`|[`AWSElasticBlockStoreVolumeSource`](#awselasticblockstorevolumesource)|AWSElasticBlockStore represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore|
|`azureDisk`|[`AzureDiskVolumeSource`](#azurediskvolumesource)|AzureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.|
|`azureFile`|[`AzureFileVolumeSource`](#azurefilevolumesource)|AzureFile represents an Azure File Service mount on the host and bind mount to the pod.|
|`cephfs`|[`CephFSVolumeSource`](#cephfsvolumesource)|CephFS represents a Ceph FS mount on the host that shares a pod's lifetime|
|`cinder`|[`CinderVolumeSource`](#cindervolumesource)|Cinder represents a cinder volume attached and mounted on kubelets host machine. More info: https://examples.k8s.io/mysql-cinder-pd/README.md|
|`configMap`|[`ConfigMapVolumeSource`](#configmapvolumesource)|ConfigMap represents a configMap that should populate this volume|
|`csi`|[`CSIVolumeSource`](#csivolumesource)|CSI (Container Storage Interface) represents ephemeral storage that is handled by certain external CSI drivers (Beta feature).|
|`downwardAPI`|[`DownwardAPIVolumeSource`](#downwardapivolumesource)|DownwardAPI represents downward API about the pod that should populate this volume|
|`emptyDir`|[`EmptyDirVolumeSource`](#emptydirvolumesource)|EmptyDir represents a temporary directory that shares a pod's lifetime. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir|
|`ephemeral`|[`EphemeralVolumeSource`](#ephemeralvolumesource)|Ephemeral represents a volume that is handled by a cluster storage driver. The volume's lifecycle is tied to the pod that defines it - it will be created before the pod starts, and deleted when the pod is removed.Use this if: a) the volume is only needed while the pod runs, b) features of normal volumes like restoring from snapshot or capacity   tracking are needed,c) the storage driver is specified through a storage class, and d) the storage driver supports dynamic volume provisioning through   a PersistentVolumeClaim (see EphemeralVolumeSource for more   information on the connection between this volume type   and PersistentVolumeClaim).Use PersistentVolumeClaim or one of the vendor-specific APIs for volumes that persist for longer than the lifecycle of an individual pod.Use CSI for light-weight local ephemeral volumes if the CSI driver is meant to be used that way - see the documentation of the driver for more information.A pod can use both types of ephemeral volumes and persistent volumes at the same time.|
|`fc`|[`FCVolumeSource`](#fcvolumesource)|FC represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.|
|`flexVolume`|[`FlexVolumeSource`](#flexvolumesource)|FlexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin.|
|`flocker`|[`FlockerVolumeSource`](#flockervolumesource)|Flocker represents a Flocker volume attached to a kubelet's host machine. This depends on the Flocker control service being running|
|`gcePersistentDisk`|[`GCEPersistentDiskVolumeSource`](#gcepersistentdiskvolumesource)|GCEPersistentDisk represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk|
|~`gitRepo`~|~[`GitRepoVolumeSource`](#gitrepovolumesource)~|~GitRepo represents a git repository at a particular revision.~ DEPRECATED: GitRepo is deprecated. To provision a container with a git repo, mount an EmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDir into the Pod's container.|
|`glusterfs`|[`GlusterfsVolumeSource`](#glusterfsvolumesource)|Glusterfs represents a Glusterfs mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/glusterfs/README.md|
|`hostPath`|[`HostPathVolumeSource`](#hostpathvolumesource)|HostPath represents a pre-existing file or directory on the host machine that is directly exposed to the container. This is generally used for system agents or other privileged things that are allowed to see the host machine. Most containers will NOT need this. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath|
|`iscsi`|[`ISCSIVolumeSource`](#iscsivolumesource)|ISCSI represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://examples.k8s.io/volumes/iscsi/README.md|
|`name`|`string`|Volume's name. Must be a DNS_LABEL and unique within the pod. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names|
|`nfs`|[`NFSVolumeSource`](#nfsvolumesource)|NFS represents an NFS mount on the host that shares a pod's lifetime More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs|
|`persistentVolumeClaim`|[`PersistentVolumeClaimVolumeSource`](#persistentvolumeclaimvolumesource)|PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims|
|`photonPersistentDisk`|[`PhotonPersistentDiskVolumeSource`](#photonpersistentdiskvolumesource)|PhotonPersistentDisk represents a PhotonController persistent disk attached and mounted on kubelets host machine|
|`portworxVolume`|[`PortworxVolumeSource`](#portworxvolumesource)|PortworxVolume represents a portworx volume attached and mounted on kubelets host machine|
|`projected`|[`ProjectedVolumeSource`](#projectedvolumesource)|Items for all in one resources secrets, configmaps, and downward API|
|`quobyte`|[`QuobyteVolumeSource`](#quobytevolumesource)|Quobyte represents a Quobyte mount on the host that shares a pod's lifetime|
|`rbd`|[`RBDVolumeSource`](#rbdvolumesource)|RBD represents a Rados Block Device mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/rbd/README.md|
|`scaleIO`|[`ScaleIOVolumeSource`](#scaleiovolumesource)|ScaleIO represents a ScaleIO persistent volume attached and mounted on Kubernetes nodes.|
|`secret`|[`SecretVolumeSource`](#secretvolumesource)|Secret represents a secret that should populate this volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret|
|`storageos`|[`StorageOSVolumeSource`](#storageosvolumesource)|StorageOS represents a StorageOS volume attached and mounted on Kubernetes nodes.|
|`vsphereVolume`|[`VsphereVirtualDiskVolumeSource`](#vspherevirtualdiskvolumesource)|VsphereVolume represents a vSphere volume attached and mounted on kubelets host machine|

## Time

Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers.

## ObjectReference

ObjectReference contains enough information to let you inspect or modify the referred object.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiVersion`|`string`|API version of the referent.|
|`fieldPath`|`string`|If referring to a piece of an object instead of an entire object, this string should contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2]. For example, if the object reference is to a container within a pod, this would take on a value like: "spec.containers{name}" (where "name" refers to the name of the container that triggered the event) or if no container name is specified "spec.containers[2]" (container with index 2 in this pod). This syntax is chosen only to have some well-defined way of referencing a part of an object.|
|`kind`|`string`|Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds|
|`name`|`string`|Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names|
|`namespace`|`string`|Namespace of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/|
|`resourceVersion`|`string`|Specific resourceVersion to which this reference is made, if any. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency|
|`uid`|`string`|UID of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids|

## LabelSelector

A label selector is a label query over a set of resources. The result of matchLabels and matchExpressions are ANDed. An empty label selector matches all objects. A null label selector matches no objects.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`pod-gc-strategy-with-label-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy-with-label-selector.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`matchExpressions`|`Array<`[`LabelSelectorRequirement`](#labelselectorrequirement)`>`|matchExpressions is a list of label selector requirements. The requirements are ANDed.|
|`matchLabels`|`Map< string , string >`|matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.|

## IntOrString

_No description available_

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`dag-disable-failFast.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-disable-failFast.yaml)

- [`retry-backoff.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-backoff.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`retry-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container.yaml)

- [`retry-on-error.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-on-error.yaml)

- [`retry-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-script.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-with-steps.yaml)

- [`template-defaults.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-defaults.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)
</details>

## Container

A single application container that you want to run within a pod.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-artifacts.yaml)

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`arguments-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`artifact-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`artifact-repository-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-repository-ref.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`coinflip-recursive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip.yaml)

- [`conditionals-complex.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals-complex.yaml)

- [`conditionals.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals.yaml)

- [`continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/continue-on-fail.yaml)

- [`cron-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-coinflip.yaml)

- [`dag-continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-continue-on-fail.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-disable-failFast.yaml)

- [`dag-enhanced-depends.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-enhanced-depends.yaml)

- [`dag-inline-clusterworkflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-clusterworkflowtemplate.yaml)

- [`dag-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflow.yaml)

- [`dag-inline-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflowtemplate.yaml)

- [`dag-multiroot.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-nested.yaml)

- [`dag-targets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-targets.yaml)

- [`dag-task-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-task-level-timeout.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`default-pdb-support.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/default-pdb-support.yaml)

- [`dns-config.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dns-config.yaml)

- [`exit-code-output-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-code-output-variable.yaml)

- [`exit-handler-dag-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-dag-level.yaml)

- [`exit-handler-slack.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-slack.yaml)

- [`exit-handler-step-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-step-level.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`exit-handlers.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handlers.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`forever.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/forever.yaml)

- [`fun-with-gifs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/gc-ttl.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)

- [`global-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)

- [`hello-hybrid.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-hybrid.yaml)

- [`hello-windows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-windows.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-world.yaml)

- [`image-pull-secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`init-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-s3.yaml)

- [`k8s-owner-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-owner-reference.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`key-only-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/key-only-artifact.yaml)

- [`label-value-from-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/label-value-from-workflow.yaml)

- [`life-cycle-hooks-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-tmpl-level.yaml)

- [`life-cycle-hooks-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-wf-level.yaml)

- [`loops-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-dag.yaml)

- [`loops-maps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-sequence.yaml)

- [`loops.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`node-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/node-selector.yaml)

- [`output-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-gcs.yaml)

- [`output-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-gc-strategy-with-label-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy-with-label-selector.yaml)

- [`pod-gc-strategy.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy.yaml)

- [`pod-metadata-wf-field.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata-wf-field.yaml)

- [`pod-metadata.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-yaml-patch.yaml)

- [`resubmit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/resubmit.yaml)

- [`retry-backoff.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-backoff.yaml)

- [`retry-container-to-completion.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container.yaml)

- [`retry-on-error.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-on-error.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-python.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)

- [`selected-executor-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/selected-executor-workflow.yaml)

- [`sidecar-dind.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar.yaml)

- [`status-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/status-reference.yaml)

- [`step-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/step-level-timeout.yaml)

- [`steps-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps-inline-workflow.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)

- [`suspend-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template.yaml)

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`synchronization-mutex-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-wf-level.yaml)

- [`template-defaults.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-defaults.yaml)

- [`template-on-exit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-on-exit.yaml)

- [`timeouts-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-pvc.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)

- [`event-consumer-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-event-binding/event-consumer-workflowtemplate.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)

- [`workflow-archive-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-archive-logs.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`args`|`Array< string >`|Arguments to the entrypoint. The docker image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell|
|`command`|`Array< string >`|Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell|
|`env`|`Array<`[`EnvVar`](#envvar)`>`|List of environment variables to set in the container. Cannot be updated.|
|`envFrom`|`Array<`[`EnvFromSource`](#envfromsource)`>`|List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.|
|`image`|`string`|Docker image name. More info: https://kubernetes.io/docs/concepts/containers/images This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets.|
|`imagePullPolicy`|`string`|Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-imagesPossible enum values: - `"Always"` means that kubelet always attempts to pull the latest image. Container will fail If the pull fails. - `"IfNotPresent"` means that kubelet pulls if the image isn't present on disk. Container will fail if the image isn't present and the pull fails. - `"Never"` means that kubelet never pulls an image, but only uses a local image. Container will fail if the image isn't present|
|`lifecycle`|[`Lifecycle`](#lifecycle)|Actions that the management system should take in response to container lifecycle events. Cannot be updated.|
|`livenessProbe`|[`Probe`](#probe)|Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`name`|`string`|Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.|
|`ports`|`Array<`[`ContainerPort`](#containerport)`>`|List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated.|
|`readinessProbe`|[`Probe`](#probe)|Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`resources`|[`ResourceRequirements`](#resourcerequirements)|Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/|
|`securityContext`|[`SecurityContext`](#securitycontext)|SecurityContext defines the security options the container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext. More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/|
|`startupProbe`|[`Probe`](#probe)|StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully. If this probe fails, the Pod will be restarted, just as if the livenessProbe failed. This can be used to provide different probe parameters at the beginning of a Pod's lifecycle, when it might take a long time to load data or warm a cache, than during steady-state operation. This cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`stdin`|`boolean`|Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false.|
|`stdinOnce`|`boolean`|Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false|
|`terminationMessagePath`|`string`|Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated.|
|`terminationMessagePolicy`|`string`|Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated.Possible enum values: - `"FallbackToLogsOnError"` will read the most recent contents of the container logs for the container status message when the container exits with an error and the terminationMessagePath has no contents. - `"File"` is the default behavior and will set the container status message to the contents of the container's terminationMessagePath when the container exits.|
|`tty`|`boolean`|Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false.|
|`volumeDevices`|`Array<`[`VolumeDevice`](#volumedevice)`>`|volumeDevices is the list of block devices to be used by the container.|
|`volumeMounts`|`Array<`[`VolumeMount`](#volumemount)`>`|Pod volumes to mount into the container's filesystem. Cannot be updated.|
|`workingDir`|`string`|Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.|

## ConfigMapKeySelector

Selects a key from a ConfigMap.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|The key to select.|
|`name`|`string`|Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names|
|`optional`|`boolean`|Specify whether the ConfigMap or its key must be defined|

## VolumeMount

VolumeMount describes a mounting of a Volume within a container.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`fun-with-gifs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fun-with-gifs.yaml)

- [`init-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/init-container.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)

- [`volumes-emptydir.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-pvc.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`mountPath`|`string`|Path within the container at which the volume should be mounted.  Must not contain ':'.|
|`mountPropagation`|`string`|mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10.|
|`name`|`string`|This must match the Name of a Volume.|
|`readOnly`|`boolean`|Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.|
|`subPath`|`string`|Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).|
|`subPathExpr`|`string`|Expanded path within the volume from which the container's volume should be mounted. Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment. Defaults to "" (volume's root). SubPathExpr and SubPath are mutually exclusive.|

## EnvVar

EnvVar represents an environment variable present in a Container.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`colored-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/colored-logs.yaml)

- [`expression-destructure-json.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-destructure-json.yaml)

- [`expression-reusing-verbose-snippets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-reusing-verbose-snippets.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)

- [`sidecar-dind.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-dind.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|Name of the environment variable. Must be a C_IDENTIFIER.|
|`value`|`string`|Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "".|
|`valueFrom`|[`EnvVarSource`](#envvarsource)|Source for the environment variable's value. Cannot be used if value is not empty.|

## EnvFromSource

EnvFromSource represents the source of a set of ConfigMaps

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`configMapRef`|[`ConfigMapEnvSource`](#configmapenvsource)|The ConfigMap to select from|
|`prefix`|`string`|An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.|
|`secretRef`|[`SecretEnvSource`](#secretenvsource)|The Secret to select from|

## Lifecycle

Lifecycle describes actions that the management system should take in response to container lifecycle events. For the PostStart and PreStop lifecycle handlers, management of the container blocks until the action is complete, unless the container process fails, in which case the handler is aborted.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`postStart`|[`LifecycleHandler`](#lifecyclehandler)|PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks|
|`preStop`|[`LifecycleHandler`](#lifecyclehandler)|PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc. The handler is not called if the container crashes or exits. The Pod's termination grace period countdown begins before the PreStop hook is executed. Regardless of the outcome of the handler, the container will eventually terminate within the Pod's termination grace period (unless delayed by finalizers). Other management of the container blocks until the hook completes or until the termination grace period is reached. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks|

## Probe

Probe describes a health check to be performed against a container to determine whether it is alive or ready to receive traffic.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`exec`|[`ExecAction`](#execaction)|Exec specifies the action to take.|
|`failureThreshold`|`integer`|Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.|
|`grpc`|[`GRPCAction`](#grpcaction)|GRPC specifies an action involving a GRPC port. This is an alpha field and requires enabling GRPCContainerProbe feature gate.|
|`httpGet`|[`HTTPGetAction`](#httpgetaction)|HTTPGet specifies the http request to perform.|
|`initialDelaySeconds`|`integer`|Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`periodSeconds`|`integer`|How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.|
|`successThreshold`|`integer`|Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.|
|`tcpSocket`|[`TCPSocketAction`](#tcpsocketaction)|TCPSocket specifies an action involving a TCP port.|
|`terminationGracePeriodSeconds`|`integer`|Optional duration in seconds the pod needs to terminate gracefully upon probe failure. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this value overrides the value provided by the pod spec. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate. Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.|
|`timeoutSeconds`|`integer`|Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|

## ContainerPort

ContainerPort represents a network port in a single container.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`containerPort`|`integer`|Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.|
|`hostIP`|`string`|What host IP to bind the external port to.|
|`hostPort`|`integer`|Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this.|
|`name`|`string`|If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.|
|`protocol`|`string`|Protocol for port. Must be UDP, TCP, or SCTP. Defaults to "TCP".Possible enum values: - `"SCTP"` is the SCTP protocol. - `"TCP"` is the TCP protocol. - `"UDP"` is the UDP protocol.|

## ResourceRequirements

ResourceRequirements describes the compute resource requirements.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`dns-config.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dns-config.yaml)

- [`fun-with-gifs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fun-with-gifs.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-yaml-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-yaml-patch.yaml)

- [`volumes-pvc.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-pvc.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`limits`|[`Quantity`](#quantity)|Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/|
|`requests`|[`Quantity`](#quantity)|Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/|

## SecurityContext

SecurityContext holds security configuration that will be applied to a container. Some fields are present in both SecurityContext and PodSecurityContext.  When both are set, the values in SecurityContext take precedence.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`sidecar-dind.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-dind.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`allowPrivilegeEscalation`|`boolean`|AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process. AllowPrivilegeEscalation is true always when the container is: 1) run as Privileged 2) has CAP_SYS_ADMIN Note that this field cannot be set when spec.os.name is windows.|
|`capabilities`|[`Capabilities`](#capabilities)|The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows.|
|`privileged`|`boolean`|Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false. Note that this field cannot be set when spec.os.name is windows.|
|`procMount`|`string`|procMount denotes the type of proc mount to use for the containers. The default is DefaultProcMount which uses the container runtime defaults for readonly paths and masked paths. This requires the ProcMountType feature flag to be enabled. Note that this field cannot be set when spec.os.name is windows.|
|`readOnlyRootFilesystem`|`boolean`|Whether this container has a read-only root filesystem. Default is false. Note that this field cannot be set when spec.os.name is windows.|
|`runAsGroup`|`integer`|The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows.|
|`runAsNonRoot`|`boolean`|Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.|
|`runAsUser`|`integer`|The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows.|
|`seLinuxOptions`|[`SELinuxOptions`](#selinuxoptions)|The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows.|
|`seccompProfile`|[`SeccompProfile`](#seccompprofile)|The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options. Note that this field cannot be set when spec.os.name is windows.|
|`windowsOptions`|[`WindowsSecurityContextOptions`](#windowssecuritycontextoptions)|The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is linux.|

## VolumeDevice

volumeDevice describes a mapping of a raw block device within a container.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`devicePath`|`string`|devicePath is the path inside of the container that the device will be mapped to.|
|`name`|`string`|name must match the name of a persistentVolumeClaim in the pod|

## SecretKeySelector

SecretKeySelector selects a key of a Secret.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`input-artifact-git.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-git.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|The key of the secret to select from.  Must be a valid secret key.|
|`name`|`string`|Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names|
|`optional`|`boolean`|Specify whether the Secret or its key must be defined|

## ManagedFieldsEntry

ManagedFieldsEntry is a workflow-id, a FieldSet and the group version of the resource that the fieldset applies to.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiVersion`|`string`|APIVersion defines the version of this resource that this field set applies to. The format is "group/version" just like the top-level APIVersion field. It is necessary to track the version of a field set because it cannot be automatically converted.|
|`fieldsType`|`string`|FieldsType is the discriminator for the different fields format and version. There is currently only one possible value: "FieldsV1"|
|`fieldsV1`|[`FieldsV1`](#fieldsv1)|FieldsV1 holds the first JSON version format as described in the "FieldsV1" type.|
|`manager`|`string`|Manager is an identifier of the workflow managing these fields.|
|`operation`|`string`|Operation is the type of operation which lead to this ManagedFieldsEntry being created. The only valid values for this field are 'Apply' and 'Update'.|
|`subresource`|`string`|Subresource is the name of the subresource used to update that object, or empty string if the object was updated through the main resource. The value of this field is used to distinguish between managers, even if they share the same name. For example, a status update will be distinct from a regular update using the same manager name. Note that the APIVersion field is not related to the Subresource field and it always corresponds to the version of the main resource.|
|`time`|[`Time`](#time)|Time is timestamp of when these fields were set. It should always be empty if Operation is 'Apply'|

## OwnerReference

OwnerReference contains enough information to let you identify an owning object. An owning object must be in the same namespace as the dependent, or be cluster-scoped, so there is no namespace field.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiVersion`|`string`|API version of the referent.|
|`blockOwnerDeletion`|`boolean`|If true, AND if the owner has the "foregroundDeletion" finalizer, then the owner cannot be deleted from the key-value store until this reference is removed. Defaults to false. To set this field, a user needs "delete" permission of the owner, otherwise 422 (Unprocessable Entity) will be returned.|
|`controller`|`boolean`|If true, this reference points to the managing controller.|
|`kind`|`string`|Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds|
|`name`|`string`|Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names|
|`uid`|`string`|UID of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#uids|

## NodeAffinity

Node affinity is a group of node affinity scheduling rules.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`preferredDuringSchedulingIgnoredDuringExecution`|`Array<`[`PreferredSchedulingTerm`](#preferredschedulingterm)`>`|The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.|
|`requiredDuringSchedulingIgnoredDuringExecution`|[`NodeSelector`](#nodeselector)|If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.|

## PodAffinity

Pod affinity is a group of inter pod affinity scheduling rules.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`preferredDuringSchedulingIgnoredDuringExecution`|`Array<`[`WeightedPodAffinityTerm`](#weightedpodaffinityterm)`>`|The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.|
|`requiredDuringSchedulingIgnoredDuringExecution`|`Array<`[`PodAffinityTerm`](#podaffinityterm)`>`|If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.|

## PodAntiAffinity

Pod anti affinity is a group of inter pod anti affinity scheduling rules.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`preferredDuringSchedulingIgnoredDuringExecution`|`Array<`[`WeightedPodAffinityTerm`](#weightedpodaffinityterm)`>`|The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.|
|`requiredDuringSchedulingIgnoredDuringExecution`|`Array<`[`PodAffinityTerm`](#podaffinityterm)`>`|If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.|

## PodDNSConfigOption

PodDNSConfigOption defines DNS resolver options of a pod.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`dns-config.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dns-config.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|Required.|
|`value`|`string`|_No description available_|

## SELinuxOptions

SELinuxOptions are the labels to be applied to the container

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`level`|`string`|Level is SELinux level label that applies to the container.|
|`role`|`string`|Role is a SELinux role label that applies to the container.|
|`type`|`string`|Type is a SELinux type label that applies to the container.|
|`user`|`string`|User is a SELinux user label that applies to the container.|

## SeccompProfile

SeccompProfile defines a pod/container's seccomp profile settings. Only one profile source may be set.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`localhostProfile`|`string`|localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work. Must be a descending path, relative to the kubelet's configured seccomp profile location. Must only be set if type is "Localhost".|
|`type`|`string`|type indicates which kind of seccomp profile will be applied. Valid options are:Localhost - a profile defined in a file on the node should be used. RuntimeDefault - the container runtime default profile should be used. Unconfined - no profile should be applied.Possible enum values: - `"Localhost"` indicates a profile defined in a file on the node should be used. The file's location relative to <kubelet-root-dir>/seccomp. - `"RuntimeDefault"` represents the default container runtime seccomp profile. - `"Unconfined"` indicates no seccomp profile is applied (A.K.A. unconfined).|

## Sysctl

Sysctl defines a kernel parameter to be set

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|Name of a property to set|
|`value`|`string`|Value of a property to set|

## WindowsSecurityContextOptions

WindowsSecurityContextOptions contain Windows-specific options and credentials.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`gmsaCredentialSpec`|`string`|GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field.|
|`gmsaCredentialSpecName`|`string`|GMSACredentialSpecName is the name of the GMSA credential spec to use.|
|`hostProcess`|`boolean`|HostProcess determines if a container should be run as a 'Host Process' container. This field is alpha-level and will only be honored by components that enable the WindowsHostProcessContainers feature flag. Setting this field without the feature flag will result in errors when validating the Pod. All of a Pod's containers must have the same effective HostProcess value (it is not allowed to have a mix of HostProcess containers and non-HostProcess containers).  In addition, if HostProcess is true then HostNetwork must also be set to true.|
|`runAsUserName`|`string`|The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.|

## PersistentVolumeClaimSpec

PersistentVolumeClaimSpec describes the common attributes of storage devices and allows a Source for provider-specific attributes

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-artifacts.yaml)

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`arguments-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-disable-archive.yaml)

- [`artifact-passing-subpath.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing-subpath.yaml)

- [`artifact-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`artifact-repository-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-repository-ref.yaml)

- [`artifactory-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifactory-artifact.yaml)

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`ci-output-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci-output-artifact.yaml)

- [`ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`workflow-template-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cluster-workflow-template/workflow-template-ref.yaml)

- [`coinflip-recursive.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/coinflip.yaml)

- [`colored-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/colored-logs.yaml)

- [`conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-artifacts.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`conditionals-complex.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals-complex.yaml)

- [`conditionals.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals.yaml)

- [`graph-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/graph-workflow.yaml)

- [`outputs-result-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/outputs-result-workflow.yaml)

- [`parallel-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/parallel-workflow.yaml)

- [`sequence-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/sequence-workflow.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/continue-on-fail.yaml)

- [`cron-backfill.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-backfill.yaml)

- [`cron-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-coinflip.yaml)

- [`dag-conditional-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-artifacts.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`dag-continue-on-fail.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-continue-on-fail.yaml)

- [`dag-custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-custom-metrics.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-disable-failFast.yaml)

- [`dag-enhanced-depends.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-enhanced-depends.yaml)

- [`dag-inline-clusterworkflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-clusterworkflowtemplate.yaml)

- [`dag-inline-cronworkflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-cronworkflow.yaml)

- [`dag-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflow.yaml)

- [`dag-inline-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-inline-workflowtemplate.yaml)

- [`dag-multiroot.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-nested.yaml)

- [`dag-targets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-targets.yaml)

- [`dag-task-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-task-level-timeout.yaml)

- [`data-transformations.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/data-transformations.yaml)

- [`default-pdb-support.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/default-pdb-support.yaml)

- [`dns-config.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dns-config.yaml)

- [`exit-code-output-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-code-output-variable.yaml)

- [`exit-handler-dag-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-dag-level.yaml)

- [`exit-handler-slack.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-slack.yaml)

- [`exit-handler-step-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-step-level.yaml)

- [`exit-handler-with-artifacts.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-artifacts.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`exit-handlers.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handlers.yaml)

- [`expression-destructure-json.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-destructure-json.yaml)

- [`expression-reusing-verbose-snippets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-reusing-verbose-snippets.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`forever.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/forever.yaml)

- [`fun-with-gifs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/gc-ttl.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)

- [`global-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`hdfs-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hdfs-artifact.yaml)

- [`hello-hybrid.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-hybrid.yaml)

- [`hello-windows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-windows.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/hello-world.yaml)

- [`http-hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-hello-world.yaml)

- [`http-success-condition.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/http-success-condition.yaml)

- [`image-pull-secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)

- [`init-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/input-artifact-s3.yaml)

- [`k8s-json-patch-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-json-patch-workflow.yaml)

- [`k8s-owner-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-owner-reference.yaml)

- [`k8s-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-patch.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`key-only-artifact.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/key-only-artifact.yaml)

- [`label-value-from-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/label-value-from-workflow.yaml)

- [`life-cycle-hooks-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-tmpl-level.yaml)

- [`life-cycle-hooks-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-wf-level.yaml)

- [`loops-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-dag.yaml)

- [`loops-maps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops-sequence.yaml)

- [`loops.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/loops.yaml)

- [`map-reduce.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/map-reduce.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`node-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/node-selector.yaml)

- [`output-artifact-gcs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-gcs.yaml)

- [`output-artifact-s3.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-gc-strategy-with-label-selector.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy-with-label-selector.yaml)

- [`pod-gc-strategy.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-gc-strategy.yaml)

- [`pod-metadata-wf-field.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata-wf-field.yaml)

- [`pod-metadata.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-yaml-patch.yaml)

- [`recursive-for-loop.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/recursive-for-loop.yaml)

- [`resubmit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/resubmit.yaml)

- [`retry-backoff.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-backoff.yaml)

- [`retry-conditional.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml)

- [`retry-container-to-completion.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-container.yaml)

- [`retry-on-error.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-on-error.yaml)

- [`retry-script.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-script.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/scripts-python.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)

- [`selected-executor-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/selected-executor-workflow.yaml)

- [`sidecar-dind.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/sidecar.yaml)

- [`status-reference.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/status-reference.yaml)

- [`step-level-timeout.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/step-level-timeout.yaml)

- [`steps-inline-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps-inline-workflow.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/steps.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)

- [`suspend-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template.yaml)

- [`synchronization-mutex-tmpl-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-tmpl-level.yaml)

- [`synchronization-mutex-wf-level.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/synchronization-mutex-wf-level.yaml)

- [`template-defaults.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-defaults.yaml)

- [`template-on-exit.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/template-on-exit.yaml)

- [`timeouts-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-pvc.yaml)

- [`work-avoidance.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/work-avoidance.yaml)

- [`event-consumer-workflowtemplate.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-event-binding/event-consumer-workflowtemplate.yaml)

- [`workflow-of-workflows.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-of-workflows.yaml)

- [`dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/steps.yaml)

- [`templates.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/templates.yaml)

- [`workflow-archive-logs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-archive-logs.yaml)

- [`workflow-template-ref-with-entrypoint-arg-passing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref-with-entrypoint-arg-passing.yaml)

- [`workflow-template-ref.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/workflow-template/workflow-template-ref.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`accessModes`|`Array< string >`|AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1|
|`dataSource`|[`TypedLocalObjectReference`](#typedlocalobjectreference)|This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source. If the AnyVolumeDataSource feature gate is enabled, this field will always have the same contents as the DataSourceRef field.|
|`dataSourceRef`|[`TypedLocalObjectReference`](#typedlocalobjectreference)|Specifies the object from which to populate the volume with data, if a non-empty volume is desired. This may be any local object from a non-empty API group (non core object) or a PersistentVolumeClaim object. When this field is specified, volume binding will only succeed if the type of the specified object matches some installed volume populator or dynamic provisioner. This field will replace the functionality of the DataSource field and as such if both fields are non-empty, they must have the same value. For backwards compatibility, both fields (DataSource and DataSourceRef) will be set to the same value automatically if one of them is empty and the other is non-empty. There are two important differences between DataSource and DataSourceRef: * While DataSource only allows two specific types of objects, DataSourceRef  allows any non-core object, as well as PersistentVolumeClaim objects.* While DataSource ignores disallowed values (dropping them), DataSourceRef  preserves all values, and generates an error if a disallowed value is  specified.(Alpha) Using this field requires the AnyVolumeDataSource feature gate to be enabled.|
|`resources`|[`ResourceRequirements`](#resourcerequirements)|Resources represents the minimum resources the volume should have. If RecoverVolumeExpansionFailure feature is enabled users are allowed to specify resource requirements that are lower than previous value but must still be higher than capacity recorded in the status field of the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources|
|`selector`|[`LabelSelector`](#labelselector)|A label query over volumes to consider for binding.|
|`storageClassName`|`string`|Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1|
|`volumeMode`|`string`|volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec.|
|`volumeName`|`string`|VolumeName is the binding reference to the PersistentVolume backing this claim.|

## PersistentVolumeClaimStatus

PersistentVolumeClaimStatus is the current status of a persistent volume claim.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`accessModes`|`Array< string >`|AccessModes contains the actual access modes the volume backing the PVC has. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1|
|`allocatedResources`|[`Quantity`](#quantity)|The storage resource within AllocatedResources tracks the capacity allocated to a PVC. It may be larger than the actual capacity when a volume expansion operation is requested. For storage quota, the larger value from allocatedResources and PVC.spec.resources is used. If allocatedResources is not set, PVC.spec.resources alone is used for quota calculation. If a volume expansion capacity request is lowered, allocatedResources is only lowered if there are no expansion operations in progress and if the actual volume capacity is equal or lower than the requested capacity. This is an alpha field and requires enabling RecoverVolumeExpansionFailure feature.|
|`capacity`|[`Quantity`](#quantity)|Represents the actual resources of the underlying volume.|
|`conditions`|`Array<`[`PersistentVolumeClaimCondition`](#persistentvolumeclaimcondition)`>`|Current Condition of persistent volume claim. If underlying persistent volume is being resized then the Condition will be set to 'ResizeStarted'.|
|`phase`|`string`|Phase represents the current phase of PersistentVolumeClaim.Possible enum values: - `"Bound"` used for PersistentVolumeClaims that are bound - `"Lost"` used for PersistentVolumeClaims that lost their underlying PersistentVolume. The claim was bound to a PersistentVolume and this volume does not exist any longer and all data on it was lost. - `"Pending"` used for PersistentVolumeClaims that are not yet bound|
|`resizeStatus`|`string`|ResizeStatus stores status of resize operation. ResizeStatus is not set by default but when expansion is complete resizeStatus is set to empty string by resize controller or kubelet. This is an alpha field and requires enabling RecoverVolumeExpansionFailure feature.|

## AWSElasticBlockStoreVolumeSource

Represents a Persistent Disk resource in AWS.An AWS EBS disk must exist before mounting to a container. The disk must also be in the same AWS zone as the kubelet. An AWS EBS disk can only be mounted as read/write once. AWS EBS volumes support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore|
|`partition`|`integer`|The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1". Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty).|
|`readOnly`|`boolean`|Specify "true" to force and set the ReadOnly property in VolumeMounts to "true". If omitted, the default is "false". More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore|
|`volumeID`|`string`|Unique ID of the persistent disk resource in AWS (Amazon EBS volume). More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore|

## AzureDiskVolumeSource

AzureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`cachingMode`|`string`|Host Caching mode: None, Read Only, Read Write.|
|`diskName`|`string`|The Name of the data disk in the blob storage|
|`diskURI`|`string`|The URI the data disk in the blob storage|
|`fsType`|`string`|Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.|
|`kind`|`string`|Expected values Shared: multiple blob disks per storage account  Dedicated: single blob disk per storage account  Managed: azure managed data disk (only in managed availability set). defaults to shared|
|`readOnly`|`boolean`|Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.|

## AzureFileVolumeSource

AzureFile represents an Azure File Service mount on the host and bind mount to the pod.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`readOnly`|`boolean`|Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.|
|`secretName`|`string`|the name of secret that contains Azure Storage Account Name and Key|
|`shareName`|`string`|Share Name|

## CephFSVolumeSource

Represents a Ceph Filesystem mount that lasts the lifetime of a pod Cephfs volumes do not support ownership management or SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`monitors`|`Array< string >`|Required: Monitors is a collection of Ceph monitors More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it|
|`path`|`string`|Optional: Used as the mounted root, rather than the full Ceph tree, default is /|
|`readOnly`|`boolean`|Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it|
|`secretFile`|`string`|Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|Optional: SecretRef is reference to the authentication secret for User, default is empty. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it|
|`user`|`string`|Optional: User is the rados user name, default is admin More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it|

## CinderVolumeSource

Represents a cinder volume resource in Openstack. A Cinder volume must exist before mounting to a container. The volume must also be in the same region as the kubelet. Cinder volumes support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://examples.k8s.io/mysql-cinder-pd/README.md|
|`readOnly`|`boolean`|Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/mysql-cinder-pd/README.md|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|Optional: points to a secret object containing parameters used to connect to OpenStack.|
|`volumeID`|`string`|volume id used to identify the volume in cinder. More info: https://examples.k8s.io/mysql-cinder-pd/README.md|

## ConfigMapVolumeSource

Adapts a ConfigMap into a volume.The contents of the target ConfigMap's Data field will be presented in a volume as files using the keys in the Data field as the file names, unless the items element is populated with specific mappings of keys to paths. ConfigMap volumes support ownership management and SELinux relabeling.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`defaultMode`|`integer`|Optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.|
|`items`|`Array<`[`KeyToPath`](#keytopath)`>`|If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.|
|`name`|`string`|Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names|
|`optional`|`boolean`|Specify whether the ConfigMap or its keys must be defined|

## CSIVolumeSource

Represents a source location of a volume to mount, managed by an external CSI driver

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`driver`|`string`|Driver is the name of the CSI driver that handles this volume. Consult with your admin for the correct name as registered in the cluster.|
|`fsType`|`string`|Filesystem type to mount. Ex. "ext4", "xfs", "ntfs". If not provided, the empty value is passed to the associated CSI driver which will determine the default filesystem to apply.|
|`nodePublishSecretRef`|[`LocalObjectReference`](#localobjectreference)|NodePublishSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls. This field is optional, and  may be empty if no secret is required. If the secret object contains more than one secret, all secret references are passed.|
|`readOnly`|`boolean`|Specifies a read-only configuration for the volume. Defaults to false (read/write).|
|`volumeAttributes`|`Map< string , string >`|VolumeAttributes stores driver-specific properties that are passed to the CSI driver. Consult your driver's documentation for supported values.|

## DownwardAPIVolumeSource

DownwardAPIVolumeSource represents a volume containing downward API info. Downward API volumes support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`defaultMode`|`integer`|Optional: mode bits to use on created files by default. Must be a Optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.|
|`items`|`Array<`[`DownwardAPIVolumeFile`](#downwardapivolumefile)`>`|Items is a list of downward API volume file|

## EmptyDirVolumeSource

Represents an empty directory for a pod. Empty directory volumes support ownership management and SELinux relabeling.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`init-container.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/init-container.yaml)

- [`volumes-emptydir.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-emptydir.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`medium`|`string`|What type of storage medium should back this directory. The default is "" which means to use the node's default medium. Must be an empty string (default) or Memory. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir|
|`sizeLimit`|[`Quantity`](#quantity)|Total amount of local storage required for this EmptyDir volume. The size limit is also applicable for memory medium. The maximum usage on memory medium EmptyDir would be the minimum value between the SizeLimit specified here and the sum of memory limits of all containers in a pod. The default is nil which means that the limit is undefined. More info: http://kubernetes.io/docs/user-guide/volumes#emptydir|

## EphemeralVolumeSource

Represents an ephemeral volume that is handled by a normal storage driver.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`volumeClaimTemplate`|[`PersistentVolumeClaimTemplate`](#persistentvolumeclaimtemplate)|Will be used to create a stand-alone PVC to provision the volume. The pod in which this EphemeralVolumeSource is embedded will be the owner of the PVC, i.e. the PVC will be deleted together with the pod.  The name of the PVC will be `<pod name>-<volume name>` where `<volume name>` is the name from the `PodSpec.Volumes` array entry. Pod validation will reject the pod if the concatenated name is not valid for a PVC (for example, too long).An existing PVC with that name that is not owned by the pod will *not* be used for the pod to avoid using an unrelated volume by mistake. Starting the pod is then blocked until the unrelated PVC is removed. If such a pre-created PVC is meant to be used by the pod, the PVC has to updated with an owner reference to the pod once the pod exists. Normally this should not be necessary, but it may be useful when manually reconstructing a broken cluster.This field is read-only and no changes will be made by Kubernetes to the PVC after it has been created.Required, must not be nil.|

## FCVolumeSource

Represents a Fibre Channel volume. Fibre Channel volumes can only be mounted as read/write once. Fibre Channel volumes support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.|
|`lun`|`integer`|Optional: FC target lun number|
|`readOnly`|`boolean`|Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.|
|`targetWWNs`|`Array< string >`|Optional: FC target worldwide names (WWNs)|
|`wwids`|`Array< string >`|Optional: FC volume world wide identifiers (wwids) Either wwids or combination of targetWWNs and lun must be set, but not both simultaneously.|

## FlexVolumeSource

FlexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`driver`|`string`|Driver is the name of the driver to use for this volume.|
|`fsType`|`string`|Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". The default filesystem depends on FlexVolume script.|
|`options`|`Map< string , string >`|Optional: Extra command options if any.|
|`readOnly`|`boolean`|Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|Optional: SecretRef is reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified. If the secret object contains more than one secret, all secrets are passed to the plugin scripts.|

## FlockerVolumeSource

Represents a Flocker volume mounted by the Flocker agent. One and only one of datasetName and datasetUUID should be set. Flocker volumes do not support ownership management or SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`datasetName`|`string`|Name of the dataset stored as metadata -> name on the dataset for Flocker should be considered as deprecated|
|`datasetUUID`|`string`|UUID of the dataset. This is unique identifier of a Flocker dataset|

## GCEPersistentDiskVolumeSource

Represents a Persistent Disk resource in Google Compute Engine.A GCE PD must exist before mounting to a container. The disk must also be in the same GCE project and zone as the kubelet. A GCE PD can only be mounted as read/write once or read-only many times. GCE PDs support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk|
|`partition`|`integer`|The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1". Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty). More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk|
|`pdName`|`string`|Unique name of the PD resource in GCE. Used to identify the disk in GCE. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk|
|`readOnly`|`boolean`|ReadOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk|

## GitRepoVolumeSource

Represents a volume that is populated with the contents of a git repository. Git repo volumes do not support ownership management. Git repo volumes support SELinux relabeling.DEPRECATED: GitRepo is deprecated. To provision a container with a git repo, mount an EmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDir into the Pod's container.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`directory`|`string`|Target directory name. Must not contain or start with '..'.  If '.' is supplied, the volume directory will be the git repository.  Otherwise, if specified, the volume will contain the git repository in the subdirectory with the given name.|
|`repository`|`string`|Repository URL|
|`revision`|`string`|Commit hash for the specified revision.|

## GlusterfsVolumeSource

Represents a Glusterfs mount that lasts the lifetime of a pod. Glusterfs volumes do not support ownership management or SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`endpoints`|`string`|EndpointsName is the endpoint name that details Glusterfs topology. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod|
|`path`|`string`|Path is the Glusterfs volume path. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod|
|`readOnly`|`boolean`|ReadOnly here will force the Glusterfs volume to be mounted with read-only permissions. Defaults to false. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod|

## HostPathVolumeSource

Represents a host path mapped into a pod. Host path volumes do not support ownership management or SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`path`|`string`|Path of the directory on the host. If the path is a symlink, it will follow the link to the real path. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath|
|`type`|`string`|Type for HostPath Volume Defaults to "" More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath|

## ISCSIVolumeSource

Represents an ISCSI disk. ISCSI volumes can only be mounted as read/write once. ISCSI volumes support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`chapAuthDiscovery`|`boolean`|whether support iSCSI Discovery CHAP authentication|
|`chapAuthSession`|`boolean`|whether support iSCSI Session CHAP authentication|
|`fsType`|`string`|Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi|
|`initiatorName`|`string`|Custom iSCSI Initiator Name. If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface <target portal>:<volume name> will be created for the connection.|
|`iqn`|`string`|Target iSCSI Qualified Name.|
|`iscsiInterface`|`string`|iSCSI Interface Name that uses an iSCSI transport. Defaults to 'default' (tcp).|
|`lun`|`integer`|iSCSI Target Lun number.|
|`portals`|`Array< string >`|iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).|
|`readOnly`|`boolean`|ReadOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false.|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|CHAP Secret for iSCSI target and initiator authentication|
|`targetPortal`|`string`|iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).|

## NFSVolumeSource

Represents an NFS mount that lasts the lifetime of a pod. NFS volumes do not support ownership management or SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`path`|`string`|Path that is exported by the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs|
|`readOnly`|`boolean`|ReadOnly here will force the NFS export to be mounted with read-only permissions. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs|
|`server`|`string`|Server is the hostname or IP address of the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs|

## PersistentVolumeClaimVolumeSource

PersistentVolumeClaimVolumeSource references the user's PVC in the same namespace. This volume finds the bound PV and mounts that volume for the pod. A PersistentVolumeClaimVolumeSource is, essentially, a wrapper around another type of volume that is owned by someone else (the system).

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`volumes-existing.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/volumes-existing.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`claimName`|`string`|ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims|
|`readOnly`|`boolean`|Will force the ReadOnly setting in VolumeMounts. Default false.|

## PhotonPersistentDiskVolumeSource

Represents a Photon Controller persistent disk resource.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.|
|`pdID`|`string`|ID that identifies Photon Controller persistent disk|

## PortworxVolumeSource

PortworxVolumeSource represents a Portworx volume resource.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|FSType represents the filesystem type to mount Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs". Implicitly inferred to be "ext4" if unspecified.|
|`readOnly`|`boolean`|Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.|
|`volumeID`|`string`|VolumeID uniquely identifies a Portworx volume|

## ProjectedVolumeSource

Represents a projected volume source

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`defaultMode`|`integer`|Mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.|
|`sources`|`Array<`[`VolumeProjection`](#volumeprojection)`>`|list of volume projections|

## QuobyteVolumeSource

Represents a Quobyte mount that lasts the lifetime of a pod. Quobyte volumes do not support ownership management or SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`group`|`string`|Group to map volume access to Default is no group|
|`readOnly`|`boolean`|ReadOnly here will force the Quobyte volume to be mounted with read-only permissions. Defaults to false.|
|`registry`|`string`|Registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes|
|`tenant`|`string`|Tenant owning the given Quobyte volume in the Backend Used with dynamically provisioned Quobyte volumes, value is set by the plugin|
|`user`|`string`|User to map volume access to Defaults to serivceaccount user|
|`volume`|`string`|Volume is a string that references an already created Quobyte volume by name.|

## RBDVolumeSource

Represents a Rados Block Device mount that lasts the lifetime of a pod. RBD volumes support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#rbd|
|`image`|`string`|The rados image name. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|
|`keyring`|`string`|Keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|
|`monitors`|`Array< string >`|A collection of Ceph monitors. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|
|`pool`|`string`|The rados pool name. Default is rbd. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|
|`readOnly`|`boolean`|ReadOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|SecretRef is name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|
|`user`|`string`|The rados user name. Default is admin. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|

## ScaleIOVolumeSource

ScaleIOVolumeSource represents a persistent ScaleIO volume

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Default is "xfs".|
|`gateway`|`string`|The host address of the ScaleIO API Gateway.|
|`protectionDomain`|`string`|The name of the ScaleIO Protection Domain for the configured storage.|
|`readOnly`|`boolean`|Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|SecretRef references to the secret for ScaleIO user and other sensitive information. If this is not provided, Login operation will fail.|
|`sslEnabled`|`boolean`|Flag to enable/disable SSL communication with Gateway, default false|
|`storageMode`|`string`|Indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned. Default is ThinProvisioned.|
|`storagePool`|`string`|The ScaleIO Storage Pool associated with the protection domain.|
|`system`|`string`|The name of the storage system as configured in ScaleIO.|
|`volumeName`|`string`|The name of a volume already created in the ScaleIO system that is associated with this volume source.|

## SecretVolumeSource

Adapts a Secret into a volume.The contents of the target Secret's Data field will be presented in a volume as files using the keys in the Data field as the file names. Secret volumes support ownership management and SELinux relabeling.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`defaultMode`|`integer`|Optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.|
|`items`|`Array<`[`KeyToPath`](#keytopath)`>`|If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.|
|`optional`|`boolean`|Specify whether the Secret or its keys must be defined|
|`secretName`|`string`|Name of the secret in the pod's namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret|

## StorageOSVolumeSource

Represents a StorageOS persistent volume resource.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.|
|`readOnly`|`boolean`|Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|SecretRef specifies the secret to use for obtaining the StorageOS API credentials.  If not specified, default values will be attempted.|
|`volumeName`|`string`|VolumeName is the human-readable name of the StorageOS volume.  Volume names are only unique within a namespace.|
|`volumeNamespace`|`string`|VolumeNamespace specifies the scope of the volume within StorageOS.  If no namespace is specified then the Pod's namespace will be used.  This allows the Kubernetes name scoping to be mirrored within StorageOS for tighter integration. Set VolumeName to any name to override the default behaviour. Set to "default" if you are not using namespaces within StorageOS. Namespaces that do not pre-exist within StorageOS will be created.|

## VsphereVirtualDiskVolumeSource

Represents a vSphere volume resource.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.|
|`storagePolicyID`|`string`|Storage Policy Based Management (SPBM) profile ID associated with the StoragePolicyName.|
|`storagePolicyName`|`string`|Storage Policy Based Management (SPBM) profile name.|
|`volumePath`|`string`|Path that identifies vSphere volume vmdk|

## LabelSelectorRequirement

A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|key is the label key that the selector applies to.|
|`operator`|`string`|operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.|
|`values`|`Array< string >`|values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.|

## EnvVarSource

EnvVarSource represents a source for the value of an EnvVar.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/arguments-parameters-from-configmap.yaml)

- [`artifact-path-placeholders.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/artifact-path-placeholders.yaml)

- [`conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditional-parameters.yaml)

- [`workspace-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/container-set-template/workspace-workflow.yaml)

- [`custom-metrics.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/custom-metrics.yaml)

- [`dag-conditional-parameters.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-conditional-parameters.yaml)

- [`exit-handler-with-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handler-with-param.yaml)

- [`expression-tag-template-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/expression-tag-template-workflow.yaml)

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)

- [`global-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-outputs.yaml)

- [`global-parameters-from-configmap-referenced-as-local-variable.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap-referenced-as-local-variable.yaml)

- [`global-parameters-from-configmap.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/global-parameters-from-configmap.yaml)

- [`handle-large-output-results.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/handle-large-output-results.yaml)

- [`k8s-wait-wf.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/k8s-wait-wf.yaml)

- [`nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/nested-workflow.yaml)

- [`output-parameter.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/output-parameter.yaml)

- [`parameter-aggregation-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/parameter-aggregation.yaml)

- [`pod-spec-from-previous-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-from-previous-step.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)

- [`suspend-template-outputs.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/suspend-template-outputs.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`configMapKeyRef`|[`ConfigMapKeySelector`](#configmapkeyselector)|Selects a key of a ConfigMap.|
|`fieldRef`|[`ObjectFieldSelector`](#objectfieldselector)|Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`, spec.nodeName, spec.serviceAccountName, status.hostIP, status.podIP, status.podIPs.|
|`resourceFieldRef`|[`ResourceFieldSelector`](#resourcefieldselector)|Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported.|
|`secretKeyRef`|[`SecretKeySelector`](#secretkeyselector)|Selects a key of a secret in the pod's namespace|

## ConfigMapEnvSource

ConfigMapEnvSource selects a ConfigMap to populate the environment variables with.The contents of the target ConfigMap's Data field will represent the key-value pairs as environment variables.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names|
|`optional`|`boolean`|Specify whether the ConfigMap must be defined|

## SecretEnvSource

SecretEnvSource selects a Secret to populate the environment variables with.The contents of the target Secret's Data field will represent the key-value pairs as environment variables.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names|
|`optional`|`boolean`|Specify whether the Secret must be defined|

## LifecycleHandler

LifecycleHandler defines a specific action that should be taken in a lifecycle hook. One and only one of the fields, except TCPSocket must be specified.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`exec`|[`ExecAction`](#execaction)|Exec specifies the action to take.|
|`httpGet`|[`HTTPGetAction`](#httpgetaction)|HTTPGet specifies the http request to perform.|
|`tcpSocket`|[`TCPSocketAction`](#tcpsocketaction)|Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility. There are no validation of this field and lifecycle hooks will fail in runtime when tcp handler is specified.|

## ExecAction

ExecAction describes a "run in container" action.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`command`|`Array< string >`|Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.|

## GRPCAction

_No description available_

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`port`|`integer`|Port number of the gRPC service. Number must be in the range 1 to 65535.|
|`service`|`string`|Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).If this is not specified, the default behavior is defined by gRPC.|

## HTTPGetAction

HTTPGetAction describes an action based on HTTP Get requests.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`daemon-nginx.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/daemon-step.yaml)

- [`dag-daemon-task.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dag-daemon-task.yaml)

- [`influxdb-ci.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/influxdb-ci.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`host`|`string`|Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.|
|`httpHeaders`|`Array<`[`HTTPHeader`](#httpheader)`>`|Custom headers to set in the request. HTTP allows repeated headers.|
|`path`|`string`|Path to access on the HTTP server.|
|`port`|[`IntOrString`](#intorstring)|Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.|
|`scheme`|`string`|Scheme to use for connecting to the host. Defaults to HTTP.Possible enum values: - `"HTTP"` means that the scheme used will be http:// - `"HTTPS"` means that the scheme used will be https://|

## TCPSocketAction

TCPSocketAction describes an action based on opening a socket

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`host`|`string`|Optional: Host name to connect to, defaults to the pod IP.|
|`port`|[`IntOrString`](#intorstring)|Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.|

## Quantity

Quantity is a fixed-point representation of a number. It provides convenient marshaling/unmarshaling in JSON and YAML, in addition to String() and AsInt64() accessors.The serialization format is:<quantity>        ::= <signedNumber><suffix>  (Note that <suffix> may be empty, from the "" case in <decimalSI>.)<digit>           ::= 0 | 1 | ... | 9 <digits>          ::= <digit> | <digit><digits> <number>          ::= <digits> | <digits>.<digits> | <digits>. | .<digits> <sign>            ::= "+" | "-" <signedNumber>    ::= <number> | <sign><number> <suffix>          ::= <binarySI> | <decimalExponent> | <decimalSI> <binarySI>        ::= Ki | Mi | Gi | Ti | Pi | Ei  (International System of units; See: http://physics.nist.gov/cuu/Units/binary.html)<decimalSI>       ::= m | "" | k | M | G | T | P | E  (Note that 1024 = 1Ki but 1000 = 1k; I didn't choose the capitalization.)<decimalExponent> ::= "e" <signedNumber> | "E" <signedNumber>No matter which of the three exponent forms is used, no quantity may represent a number greater than 2^63-1 in magnitude, nor may it have more than 3 decimal places. Numbers larger or more precise will be capped or rounded up. (E.g.: 0.1m will rounded up to 1m.) This may be extended in the future if we require larger or smaller quantities.When a Quantity is parsed from a string, it will remember the type of suffix it had, and will use the same type again when it is serialized.Before serializing, Quantity will be put in "canonical form". This means that Exponent/suffix will be adjusted up or down (with a corresponding increase or decrease in Mantissa) such that:  a. No precision is lost  b. No fractional digits will be emitted  c. The exponent (or suffix) is as large as possible.The sign will be omitted unless the number is negative.Examples:  1.5 will be serialized as "1500m"  1.5Gi will be serialized as "1536Mi"Note that the quantity will NEVER be internally represented by a floating point number. That is the whole point of this exercise.Non-canonical values will still parse as long as they are well formed, but will be re-emitted in their canonical form. (So always use canonical form, or don't diff.)This format is intended to make it difficult to use these numbers without writing some sort of special handling code in the hopes that that will cause implementors to also use a fixed point implementation.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`dns-config.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/dns-config.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-yaml-patch.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/pod-spec-yaml-patch.yaml)
</details>

## Capabilities

Adds and removes POSIX capabilities from running containers.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`add`|`Array< string >`|Added capabilities|
|`drop`|`Array< string >`|Removed capabilities|

## FieldsV1

FieldsV1 stores a set of fields in a data structure like a Trie, in JSON format.Each key is either a '.' representing the field itself, and will always map to an empty set, or a string representing a sub-field or item. The string will follow one of these four formats: 'f:<name>', where <name> is the name of a field in a struct, or key in a map 'v:<value>', where <value> is the exact json formatted value of a list item 'i:<index>', where <index> is position of a item in a list 'k:<keys>', where <keys> is a map of  a list item's key fields to their unique values If a key maps to an empty Fields value, the field that key represents is part of the set.The exact format is defined in sigs.k8s.io/structured-merge-diff

## PreferredSchedulingTerm

An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`preference`|[`NodeSelectorTerm`](#nodeselectorterm)|A node selector term, associated with the corresponding weight.|
|`weight`|`integer`|Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.|

## NodeSelector

A node selector represents the union of the results of one or more label queries over a set of nodes; that is, it represents the OR of the selectors represented by the node selector terms.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`nodeSelectorTerms`|`Array<`[`NodeSelectorTerm`](#nodeselectorterm)`>`|Required. A list of node selector terms. The terms are ORed.|

## WeightedPodAffinityTerm

The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`podAffinityTerm`|[`PodAffinityTerm`](#podaffinityterm)|Required. A pod affinity term, associated with the corresponding weight.|
|`weight`|`integer`|weight associated with matching the corresponding podAffinityTerm, in the range 1-100.|

## PodAffinityTerm

Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`labelSelector`|[`LabelSelector`](#labelselector)|A label query over a set of resources, in this case pods.|
|`namespaceSelector`|[`LabelSelector`](#labelselector)|A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field. null selector and null or empty namespaces list means "this pod's namespace". An empty selector ({}) matches all namespaces. This field is beta-level and is only honored when PodAffinityNamespaceSelector feature is enabled.|
|`namespaces`|`Array< string >`|namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector. null or empty namespaces list and null namespaceSelector means "this pod's namespace"|
|`topologyKey`|`string`|This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.|

## TypedLocalObjectReference

TypedLocalObjectReference contains enough information to let you locate the typed referenced object inside the same namespace.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiGroup`|`string`|APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.|
|`kind`|`string`|Kind is the type of resource being referenced|
|`name`|`string`|Name is the name of resource being referenced|

## PersistentVolumeClaimCondition

PersistentVolumeClaimCondition contails details about state of pvc

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`lastProbeTime`|[`Time`](#time)|Last time we probed the condition.|
|`lastTransitionTime`|[`Time`](#time)|Last time the condition transitioned from one status to another.|
|`message`|`string`|Human-readable message indicating details about last transition.|
|`reason`|`string`|Unique, this should be a short, machine understandable string that gives the reason for condition's last transition. If it reports "ResizeStarted" that means the underlying persistent volume is being resized.|
|`status`|`string`|_No description available_|
|`type`|`string`|Possible enum values: - `"FileSystemResizePending"` - controller resize is finished and a file system resize is pending on node - `"Resizing"` - a user trigger resize of pvc has been started|

## KeyToPath

Maps a string key to a path within a volume.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|The key to project.|
|`mode`|`integer`|Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.|
|`path`|`string`|The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.|

## DownwardAPIVolumeFile

DownwardAPIVolumeFile represents information to create the file containing the pod field

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fieldRef`|[`ObjectFieldSelector`](#objectfieldselector)|Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.|
|`mode`|`integer`|Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.|
|`path`|`string`|Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'|
|`resourceFieldRef`|[`ResourceFieldSelector`](#resourcefieldselector)|Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.|

## PersistentVolumeClaimTemplate

PersistentVolumeClaimTemplate is used to produce PersistentVolumeClaim objects as part of an EphemeralVolumeSource.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`metadata`|[`ObjectMeta`](#objectmeta)|May contain labels and annotations that will be copied into the PVC when creating it. No other fields are allowed and will be rejected during validation.|
|`spec`|[`PersistentVolumeClaimSpec`](#persistentvolumeclaimspec)|The specification for the PersistentVolumeClaim. The entire content is copied unchanged into the PVC that gets created from this template. The same fields as in a PersistentVolumeClaim are also valid here.|

## VolumeProjection

Projection that may be projected along with other supported volume types

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`configMap`|[`ConfigMapProjection`](#configmapprojection)|information about the configMap data to project|
|`downwardAPI`|[`DownwardAPIProjection`](#downwardapiprojection)|information about the downwardAPI data to project|
|`secret`|[`SecretProjection`](#secretprojection)|information about the secret data to project|
|`serviceAccountToken`|[`ServiceAccountTokenProjection`](#serviceaccounttokenprojection)|information about the serviceAccountToken data to project|

## ObjectFieldSelector

ObjectFieldSelector selects an APIVersioned field of an object.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiVersion`|`string`|Version of the schema the FieldPath is written in terms of, defaults to "v1".|
|`fieldPath`|`string`|Path of the field to select in the specified API version.|

## ResourceFieldSelector

ResourceFieldSelector represents container resources (cpu, memory) and their output format

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`containerName`|`string`|Container name: required for volumes, optional for env vars|
|`divisor`|[`Quantity`](#quantity)|Specifies the output format of the exposed resources, defaults to "1"|
|`resource`|`string`|Required: resource to select|

## HTTPHeader

HTTPHeader describes a custom header to be used in HTTP probes

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|The header field name|
|`value`|`string`|The header field value|

## NodeSelectorTerm

A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`matchExpressions`|`Array<`[`NodeSelectorRequirement`](#nodeselectorrequirement)`>`|A list of node selector requirements by node's labels.|
|`matchFields`|`Array<`[`NodeSelectorRequirement`](#nodeselectorrequirement)`>`|A list of node selector requirements by node's fields.|

## ConfigMapProjection

Adapts a ConfigMap into a projected volume.The contents of the target ConfigMap's Data field will be presented in a projected volume as files using the keys in the Data field as the file names, unless the items element is populated with specific mappings of keys to paths. Note that this is identical to a configmap volume source without the default mode.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`fibonacci-seq-conditional-param.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/fibonacci-seq-conditional-param.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`items`|`Array<`[`KeyToPath`](#keytopath)`>`|If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.|
|`name`|`string`|Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names|
|`optional`|`boolean`|Specify whether the ConfigMap or its keys must be defined|

## DownwardAPIProjection

Represents downward API info for projecting into a projected volume. Note that this is identical to a downwardAPI volume source without the default mode.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`items`|`Array<`[`DownwardAPIVolumeFile`](#downwardapivolumefile)`>`|Items is a list of DownwardAPIVolume file|

## SecretProjection

Adapts a secret into a projected volume.The contents of the target Secret's Data field will be presented in a projected volume as files using the keys in the Data field as the file names. Note that this is identical to a secret volume source without the default mode.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`buildkit-template.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/buildkit-template.yaml)

- [`secrets.yaml`](https://github.com/argoproj/argo-workflows/blob/master/examples/secrets.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`items`|`Array<`[`KeyToPath`](#keytopath)`>`|If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.|
|`name`|`string`|Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names|
|`optional`|`boolean`|Specify whether the Secret or its key must be defined|

## ServiceAccountTokenProjection

ServiceAccountTokenProjection represents a projected service account token volume. This projection can be used to insert a service account token into the pods runtime filesystem for use against APIs (Kubernetes API Server or otherwise).

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`audience`|`string`|Audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token. The audience defaults to the identifier of the apiserver.|
|`expirationSeconds`|`integer`|ExpirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token. The kubelet will start trying to rotate the token if the token is older than 80 percent of its time to live or if the token is older than 24 hours.Defaults to 1 hour and must be at least 10 minutes.|
|`path`|`string`|Path is the path relative to the mount point of the file to project the token into.|

## NodeSelectorRequirement

A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|The label key that the selector applies to.|
|`operator`|`string`|Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.Possible enum values: - `"DoesNotExist"` - `"Exists"` - `"Gt"` - `"In"` - `"Lt"` - `"NotIn"`|
|`values`|`Array< string >`|An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.|
