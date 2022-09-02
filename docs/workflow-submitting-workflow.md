# One Workflow Submitting Another

> v2.8 and after

If you want one workflow to create another, you can do this using `curl`. You'll need an [access token](access-token.md). Typically the best way is to submit from a workflow template:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: demo-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: a
            template: create-wf
    - name: create-wf
      script:
        image: curlimages/curl:latest
        command:
          - sh
        source: >
          curl https://argo-server:2746/api/v1/workflows/argo/submit \
            -fs \
            -H "Authorization: Bearer eyJhbGci..." \
            -d '{"resourceKind": "WorkflowTemplate", "resourceName": "wait", "submitOptions": {"labels": "workflows.argoproj.io/workflow-template=wait"}}'
```
