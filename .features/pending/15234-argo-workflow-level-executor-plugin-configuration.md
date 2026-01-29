Description: Add Optional Argo Workflow–Level Configuration for Executor Plugins
Author: [ntny](https://github.com/ntny)
Component: General
Issues: 15234

This PR allows configuring the Argo Workflow Executor Plugin for a specific Argo Workflow directly within the Workflow spec.
This feature enable via `ARGO_WORKFLOW_LEVEL_EXECUTOR_PLUGINS=true` controller env variable


    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: workflow-controller
    spec:
      template:
        spec:
          containers:
            - name: workflow-controller
              env:
                - name: ARGO_WORKFLOW_LEVEL_EXECUTOR_PLUGINS
                  value: "true"

Sample executor plugin definition in the Argo workflow spec:

    apiVersion: argoproj.io/v1alpha1
    kind: Workflow
    metadata:
      name: wf-level-plugin
      namespace: argo
    spec:
      entrypoint: hello-hello-hello
      executorPlugins:
        - spec:
            sidecar:
              container:
                name: print-message-plugin
                image: print-message-plugin:latest
                imagePullPolicy: IfNotPresent
                ports:
                  - containerPort: 8080
                resources:
                  limits:
                    cpu: 500m
                    memory: 128Mi
                  requests:
                    cpu: 250m
                    memory: 64Mi

The definition of the step that uses the plugin is the same for both global and workflow-level specs:

      templates:
        - name: hello-hello-hello
          steps:
            - - name: hello1
                template: print-message
                arguments:
                  parameters:
                    - name: message
                      value: "hello1"
        - name: print-message # step which use the plugin
          inputs:
            parameters:
              - name: message
          plugin:
            print-message-plugin:
              args: ["{{inputs.parameters.message}}"]