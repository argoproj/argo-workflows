# Asynchronous Job Pattern

## Introduction

If triggering an external job (e.g. an Amazon EMR job) from Argo that does not run to completion in a container, there are two options:

- create a container that polls the external job completion status
- combine a trigger step that starts the job with a `Suspend` step that is resumed by an API call to Argo when the external job is complete.

This document describes the second option in more detail.

## The pattern

The pattern involves two steps - the first step is a short-running step that triggers a long-running job outside Argo (e.g. an HTTP submission), and the second step is a `Suspend` step that suspends workflow execution and is ultimately either resumed or stopped (i.e. failed) via a call to the Argo API when the job outside Argo succeeds or fails.

When implemented as a `WorkflowTemplate` it can look something like this:

```yaml
# async-pattern.yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: external-job-template
spec:
  entrypoint: main
  templates:
  - name: main
    inputs:
      parameters:
        - name: "cmd"
    steps:
      - - name: trigger-job
          template: trigger-job
          arguments:
            parameters:
              - name: "cmd"
                value: "{{inputs.parameters.cmd}}"
      - - name: wait-completion
          template: wait-completion
          arguments:
            parameters:
              - name: uuid
                value: "{{steps.trigger-job.outputs.result}}"

  - name: trigger-job
    inputs:
      parameters:
        - name: "cmd"
    container:
      image: appropriate/curl:latest
      command: ["/bin/sh", "-c"]
      args: ["{{inputs.parameters.cmd}}"]

  - name: wait-completion
    inputs:
      parameters:
        - name: uuid
    suspend: {}
```

In this case the `cmd` parameter can be a command that makes an HTTP call via curl to an endpoint that returns a job UUID. More sophisticated submission and parsing of submission output could be done with something like a Python script step.

On job completion the external job would need to call either resume if successful:

You may need  an [access token](access-token.md).

### Prep
```bash
# add template to cluster
kubectl apply -f async-pattern.yaml

# configure some vars
export ARGO=https://localhost:2746
export NAMESPACE=argo
export WFNAME=async-pattern
export WFTNAME=external-job-template
export ARGO_TOKEN=<INSERT>
# some value that gets generated beforehand and used as a selector later
export UUID=cb2f8900-4e01-424f-8a26-1975068b97ed 
```

### Using Argo CLI
```bash
# submit workflow using argo CLI
argo submit -n $NAMESPACE --name $WFNAME \
  --from workflowtemplate/external-job-template -p cmd="echo $UUID"

# resume
argo resume --node-field-selector "inputs.parameters.uuid.value=$UUID" $WFNAME

# stop
argo stop --node-field-selector  "inputs.parameters.uuid.value=$UUID" $WFNAME
```

### Using cURL:
```bash
# alternatively submit via curl
cat > data.json <<EOF
{
    "resourceKind": "WorkflowTemplate",
    "resourceName": "$WFTNAME",
    "submitOptions": {
        "name": "$WFNAME",
        "labels": "workflows.argoproj.io/workflow-template=$WFTNAME",
        "parameters": [
            "cmd=echo $UUID"
        ]
    }
}
EOF
curl --request POST \
  --url $ARGO/api/v1/workflows/$NAMESPACE/submit \
  --header 'Content-Type: application/json' \
  --header "Authorization: Bearer $ARGO_TOKEN" \
  --data @./data.json
    
# resume flow
cat > resume.json <<EOF
{
  "namespace": "$NAMESPACE",
  "name": "$WFNAME",
  "nodeFieldSelector": "inputs.parameters.uuid.value=$UUID"
}
EOF
curl --request PUT \
  --url $ARGO/api/v1/workflows/$NAMESPACE/$WFNAME/resume \
  --header 'Content-Type: application/json' \
  --header "Authorization: Bearer $ARGO_TOKEN" \
  --data @./resume.json

# or stop if unsuccessful:
cat > stop.json <<EOF
{
  "namespace": "$NAMESPACE",
  "name": "$WFNAME",
  "nodeFieldSelector": "inputs.parameters.uuid.value=$UUID",
  "message": "<FAILURE-MESSAGE>"
}
EOF
curl --request PUT \
  --url $ARGO/api/v1/workflows/$NAMESPACE/$WFNAME/stop \
  --header 'content-type: application/json' \
  --header "Authorization: Bearer $ARGO_TOKEN" \
  --data @./stop.json 
```

## Retrying failed jobs

Using `argo retry` on failed jobs that follow this pattern will cause Argo to re-attempt the Suspend step without re-triggering the job.  

Instead you need to use the `--restart-successful` option, e.g. if using the template from above:

```bash
argo retry $WFNAME --restart-successful --node-field-selector templateRef.template=main,phase=Failed
```
