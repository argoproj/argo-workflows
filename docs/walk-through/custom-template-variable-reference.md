# Custom Template Variable Reference

In this example, we can see how we can use the other template language variable reference (E.g: Jinja) in Argo workflow template.
Argo will validate and resolve only the variable that starts with an Argo allowed prefix
{***"item", "steps", "inputs", "outputs", "workflow", "tasks"***}

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: custom-template-variable-
spec:
  entrypoint: hello-hello-hello

  templates:
    - name: hello-hello-hello
      steps:
        - - name: hello1
            template: whalesay
            arguments:
              parameters: [{name: message, value: "hello1"}]
        - - name: hello2a
            template: whalesay
            arguments:
              parameters: [{name: message, value: "hello2a"}]
          - name: hello2b
            template: whalesay
            arguments:
              parameters: [{name: message, value: "hello2b"}]

    - name: whalesay
      inputs:
        parameters:
          - name: message
      container:
        image: docker/whalesay
        command: [cowsay]
        args: ["{{user.username}}"]

```
