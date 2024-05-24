# Enhanced Depends Logic

> v2.9 and after

## Introduction

Previous to version 2.8, the only way to specify dependencies in DAG templates was to use the `dependencies` field and
specify a list of other tasks the current task depends on. This syntax was limiting because it does not allow the user to
specify which _result_ of the task to depend on. For example, a task may only be relevant to run if the dependent task
succeeded (or failed, etc.).

## Depends

To remedy this, there exists a new field called `depends`, which allows users to specify dependent tasks, their statuses,
as well as any complex boolean logic. The field is a `string` field and the syntax is expression-like with operands having
form `<task-name>.<task-result>`. Examples include `task-1.Succeeded`, `task-2.Failed`, `task-3.Daemoned`. The full list of
available task results is as follows:

|  Task Result | Description    | Meaning |
|:------------:|----------------|---------|
| `.Succeeded` | Task Succeeded | Task finished with no error |
| `.Failed` | Task Failed | Task exited with a non-0 exit code |
| `.Errored` | Task Errored | Task had an error other than a non-0 exit code |
| `.Skipped` | Task Skipped | Task was skipped |
| `.Omitted` | Task Omitted | Task was omitted |
| `.Daemoned` | Task is Daemoned and is not Pending | |

A tasks is considered `Skipped` if its `when` condition evaluates to false. On the other hand, if a task doesn't run
because its `depends` evaluated to false it is `Omitted`.

For convenience, an omitted task result is equivalent to `(task.Succeeded || task.Skipped || task.Daemoned)`.

For example:

```yaml
depends: "task || task-2.Failed"
```

is equivalent to:

```yaml
depends: (task.Succeeded || task.Skipped || task.Daemoned) || task-2.Failed
```

Full boolean logic is also available. Operators include:

* `&&`
* `||`
* `!`

 Example:

```yaml
depends: "(task-2.Succeeded || task-2.Skipped) && !task-3.Failed"
```

In the case that you're depending on a task that uses `withItems`, you can depend on
whether any of the item tasks are successful or all have failed using `.AnySucceeded` and `.AllFailed`, for example:

```yaml
depends: "task-1.AnySucceeded || task-2.AllFailed"
```

## Compatibility with `dependencies` and `dag.task.continueOn`

This feature is fully compatible with `dependencies` and conversion is easy.

To convert simply join your `dependencies` with `&&`:

```yaml
dependencies: ["A", "B", "C"]
```

is equivalent to:

```yaml
depends: "A && B && C"
```

Because of the added control found in `depends`, the `dag.task.continueOn` is not available when using it. Furthermore,
it is not possible to use both `dependencies` and `depends` in the same task group.
