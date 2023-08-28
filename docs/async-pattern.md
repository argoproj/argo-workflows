# Asynchronous Job Pattern

## Introduction

If triggering an external job (e.g. an Amazon EMR job) from Argo that does not run to completion in a container, there are two options:

- create a container that polls the external job completion status
- combine a trigger step that starts the job with a [`suspend`](walk-through/suspending.md) step that is resumed by an API call to Argo when the external job is complete.

This document describes the second option in more detail.

## The pattern

The pattern involves two steps - the first step is a short-running step that triggers a long-running job outside Argo (e.g. an HTTP submission), and the second step is a `suspend` step that suspends workflow execution and is ultimately either resumed or stopped (i.e. failed) via a call to the Argo API when the job outside Argo succeeds or fails.

When implemented as a `WorkflowTemplate` it can look something like this:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: external-job-template
spec:
  templates:
  - name: run-external-job
    inputs:
      parameters:
        - name: "job-cmd"
    steps:
      - - name: trigger-job
          template: trigger-job
          arguments:
            parameters:
              - name: "job-cmd"
                value: "{{inputs.parameters.job-cmd}}"
      - - name: wait-completion
          template: wait-completion
          arguments:
            parameters:
              - name: uuid
                value: "{{steps.trigger-job.outputs.result}}"

  - name: trigger-job
    inputs:
      parameters:
        - name: "job-cmd"
          value: "{{inputs.parameters.job-cmd}}"
      image: appropriate/curl:latest
      command: ["/bin/sh", "-c"]
      args: ["{{inputs.parameters.cmd}}"]

  - name: wait-completion
    inputs:
      parameters:
        - name: uuid
    suspend: {}
```

In this case the ```job-cmd``` parameter can be a command that makes an HTTP call via curl to an endpoint that returns a job UUID. More sophisticated submission and parsing of submission output could be done with something like a Python script step.

On job completion the external job would need to call either resume if successful:

You may need  an [access token](access-token.md).

```bash
curl --request PUT \
  --url https://localhost:2746/api/v1/workflows/<NAMESPACE>/<WORKFLOWNAME>/resume
  --header 'content-type: application/json' \
  --header "Authorization: Bearer $ARGO_TOKEN" \
  --data '{
      "namespace": "<NAMESPACE>",
      "name": "<WORKFLOWNAME>",
      "nodeFieldSelector": "inputs.parameters.uuid.value=<UUID>"
    }'  
```

or stop if unsuccessful:

```bash
curl --request PUT \
  --url https://localhost:2746/api/v1/workflows/<NAMESPACE>/<WORKFLOWNAME>/stop
  --header 'content-type: application/json' \
  --header "Authorization: Bearer $ARGO_TOKEN" \
  --data '{
      "namespace": "<NAMESPACE>",
      "name": "<WORKFLOWNAME>",
      "nodeFieldSelector": "inputs.parameters.uuid.value=<UUID>",
      "message": "<FAILURE-MESSAGE>"
    }'  
```

## Retrying failed jobs

Using `argo retry` on failed jobs that follow this pattern will cause Argo to re-attempt the `suspend` step without re-triggering the job.

Instead you need to use the `--restart-successful` option, e.g. if using the template from above:

```bash
argo retry <WORKFLOWNAME> --restart-successful --node-field-selector templateRef.template=run-external-job,phase=Failed
```
