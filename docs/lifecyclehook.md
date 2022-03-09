# LifecycleHook

> v3.3 and after

## Introduction

A [`LifecycleHook`](https://argoproj.github.io/argo-workflows/fields/#lifecyclehook) triggers an action based on a conditional expression. It is configured either at the workflow-level or template-level, for instance as a function of the `workflow.status` or `steps.status`, respectively. A `LifecycleHook` executes during execution time and executes once.

In other words, a `LifecycleHook` functions like an [exit handler](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handlers.yaml) with a conditional expression.

**Workflow-level `LifecycleHook`**: Executes the workflow when a configured expression is met.
- [Workflow-level LifecycleHook example](https://github.com/argoproj/argo-workflows/blob/45730a9cdeb588d0e52b1ac87b6e0ca391a95a81/examples/life-cycle-hooks-wf-level.yaml)

**Template-level LifecycleHook**: Executes the template when a configured expression is met.
- [Template-level LifecycleHook example](https://github.com/argoproj/argo-workflows/blob/45730a9cdeb588d0e52b1ac87b6e0ca391a95a81/examples/life-cycle-hooks-tmpl-level.yaml)

## Supported conditions

- [Exit handler variables](https://github.com/argoproj/argo-workflows/blob/ebd3677c7a9c973b22fa81ef3b409404a38ec331/docs/variables.md#exit-handler): `workflow.status` and `workflow.failures`
- [`template`](https://argoproj.github.io/argo-workflows/fields/#template)
-  [`templateRef`](https://argoproj.github.io/argo-workflows/fields/#templateref)
- [`arguments`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals.yaml)

## Unsupported conditions

- [`outputs`](https://argoproj.github.io/argo-workflows/fields/#outputs) are not usable since `LifecycleHook` executes during execution time and `outputs` are not produced until the step is completed.

## Notification use case

A `LifecycleHook` can be used to configure a notification depending on a workflow status change or template status change, like the example below:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 generateName: lifecycle-hook-
spec:
 entrypoint: main
 hooks:
   exit:
     template: http
   running:
     expression: workflow.status == "Running"
     template: http
 templates:
   - name: main
     steps:
       - - name: step1
           template: heads

   - name: heads
     container:
       image: alpine:3.6
       command: [sh, -c]
       args: ["echo \"it was heads\""]

   - name: http
     http:
       url: http://dummy.restapiexample.com/api/v1/employees
```
