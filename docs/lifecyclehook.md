# Lifecycle-Hook

> v3.3 and after

## Introduction

A [`LifecycleHook`](fields.md#lifecyclehook) triggers an action based on a conditional expression or on completion of a step or template. It is configured either at the workflow-level or template-level, for instance as a function of the `workflow.status` or `steps.status`, respectively. A `LifecycleHook` executes during execution time and executes once. It will execute in parallel to its step or template once the expression is satisfied.

In other words, a `LifecycleHook` functions like an [exit handler](https://github.com/argoproj/argo-workflows/blob/master/examples/exit-handlers.yaml) with a conditional expression. You must not name a `LifecycleHook` `exit` or it becomes an exit handler; otherwise the hook name has no relevance.

**Workflow-level `LifecycleHook`**: Executes the template when a configured expression is met during the workflow.

- [Workflow-level Lifecycle-Hook example](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-wf-level.yaml)

**Template-level `Lifecycle-Hook`**: Executes the template when a configured expression is met during the step in which it is defined.

- [Template-level Lifecycle-Hook example](https://github.com/argoproj/argo-workflows/blob/master/examples/life-cycle-hooks-tmpl-level.yaml)

## Supported conditions

- [Exit handler variables](variables.md#exit-handler): `workflow.status` and `workflow.failures`
- [`template`](fields.md#template)
- [`templateRef`](fields.md#templateref)
- [`arguments`](https://github.com/argoproj/argo-workflows/blob/master/examples/conditionals.yaml)

## Unsupported conditions

- [`outputs`](fields.md#outputs) are not usable since `LifecycleHook` executes during execution time and `outputs` are not produced until the step is completed. You can use outputs from previous steps, just not the one you're hooking into. If you'd like to use outputs create an exit handler instead - all the status variable are available there so you can still conditionally decide what to do.

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

> Put differently, an exit handler is like a workflow-level `LifecycleHook` with an expression of `workflow.status == "Succeeded"` or `workflow.status == "Failed"` or `workflow.status == "Error"`.
