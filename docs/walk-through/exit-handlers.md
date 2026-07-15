# Exit handlers

An exit handler is a template that *always* executes, irrespective of success or failure, at the end of the workflow.

Some common use cases of exit handlers are:

- cleaning up after a workflow runs
- sending notifications of workflow status (e.g., e-mail/Slack)
- posting the pass/fail status to a web-hook result (e.g. GitHub build result)
- resubmitting or submitting another workflow

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: exit-handlers-
spec:
  entrypoint: intentional-fail
  onExit: exit-handler                  # invoke exit-handler template at end of the workflow
  templates:
  # primary workflow template
  - name: intentional-fail
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo intentional failure; exit 1"]

  # Exit handler templates
  # After the completion of the entrypoint template, the status of the
  # workflow is made available in the global variable {{workflow.status}}.
  # {{workflow.status}} will be one of: Succeeded, Failed, Error
  - name: exit-handler
    steps:
    - - name: notify
        template: send-email
      - name: celebrate
        template: celebrate
        when: "{{workflow.status}} == Succeeded"
      - name: cry
        template: cry
        when: "{{workflow.status}} != Succeeded"
  - name: send-email
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo send e-mail: {{workflow.name}} {{workflow.status}} {{workflow.duration}}"]
  - name: celebrate
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hooray!"]
  - name: cry
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo boohoo!"]
```
