# Asynchronous Job Pattern

## Introduction

To trigger an external job (for example, an Amazon EMR job) from Argo that does not run to completion in a container, you have two options:

- create a container that polls the external job completion status
- combine a trigger step that starts the job with a [`suspend`](walk-through/suspending.md) step that an API call to Argo resumes when the external job is complete.

This document describes the second option in more detail.

## The pattern

The pattern involves two steps.
The first step is a short-running step that triggers a long-running job outside Argo (for example, an HTTP submission).
The second step is a `suspend` step that suspends workflow execution.
When the job outside Argo succeeds or fails, a call to the Argo API either resumes or stops (that is, fails) the `suspend` step.

A WorkflowTemplate that implements this pattern can look like this:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: external-job-template
spec:
  entrypoint: run-external-job
  arguments:
    parameters:
      - name: "job-cmd"
  templates:
    - name: run-external-job
      inputs:
        parameters:
          - name: "job-cmd"
            value: "{{workflow.parameters.job-cmd}}"
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
      container:
        image: appropriate/curl:latest
        command: [ "/bin/sh", "-c" ]
        args: [ "{{inputs.parameters.job-cmd}}" ]

    - name: wait-completion
      inputs:
        parameters:
          - name: uuid
      suspend: { }
```

In this case, the `job-cmd` parameter can be a command that uses `curl` to make an HTTP call to an endpoint that returns a job UUID.
For more sophisticated submission and parsing of submission output, you can use a Python script step.

When the job completes, the external job must call either `resume` if successful:

```bash
curl --request PUT \
  --url https://localhost:2746/api/v1/workflows/<NAMESPACE>/<WORKFLOWNAME>/resume \
  --header 'content-type: application/json' \
  --header "Authorization: $ARGO_TOKEN" \
  --data '{
      "namespace": "<NAMESPACE>",
      "name": "<WORKFLOWNAME>",
      "nodeFieldSelector": "inputs.parameters.uuid.value=<UUID>"
    }'
```

or `stop` if unsuccessful:

```bash
curl --request PUT \
  --url https://localhost:2746/api/v1/workflows/<NAMESPACE>/<WORKFLOWNAME>/stop \
  --header 'content-type: application/json' \
  --header "Authorization: $ARGO_TOKEN" \
  --data '{
      "namespace": "<NAMESPACE>",
      "name": "<WORKFLOWNAME>",
      "nodeFieldSelector": "inputs.parameters.uuid.value=<UUID>",
      "message": "<FAILURE-MESSAGE>"
    }'
```

To perform the API call you may need an [access token](access-token.md).

## Retrying failed jobs

If you use `argo retry` on failed workflows that follow this pattern, Argo re-attempts the `suspend` step without re-triggering the job.

Instead, use the `--restart-successful` option.
For example, with the template above:

```bash
argo retry <WORKFLOWNAME> --restart-successful --node-field-selector templateName=run-external-job,phase=Failed
```
