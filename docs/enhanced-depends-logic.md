# Enhanced Depends Logic

![GA](assets/ga.svg)

> v2.9 and after

## Introduction

Previous to version 2.8, the only way to specify dependencies in DAG templates was to use the `dependencies` field and
specify a list of other tasks the current task depends on. This syntax was limiting because it does not allow the user to
specify which _result_ of the task to depend on. For example, a task may only be relevant to run if the dependent task
succeeded (or failed, etc.).

## Depends

To remedy this, there exists a new field called `depends`, which allows users to specify dependent tasks, their statuses,
as well as any complex boolean logic. The field is a `string` field and the syntax is expression-like with operands having
form `<task-name>.<task-result>`. Examples include `task-1.Suceeded`, `task-2.Failed`, `task-3.Damenoed`. The full list of
available task results is as follows:

|  Task Result | Description    |
|:------------:|----------------|
| `.Succeeded` | Task Succeeded |
| `.Failed` | Task Failed |
| `.Errored` | Task Errored |
| `.Skipped` | Task Skipped |
| `.Daemoned` | Task is Daemoned and is not Pending |

For convenience, if an omitted task result is equivalent to `(task.Succeeded || task.Skipped || task.Daemoned)`.

For example:

```
depends: "task || task-2.Failed"
```

is equivalent to:

```
depends: (task.Succeeded || task.Skipped || task.Daemoned) || task-2.Failed
```

Full boolean logic is also available. Operators include:
 
 * `&&`
 * `||`
 * `!`
 
 Example:

```
depends: "(task-2.Succeeded || task-2.Skipped) && !task-3.Failed"
```

In the case that you're depending on a task that uses withItems, you can depend on
whether any of the item tasks are successful or all have failed using .AnySucceeded and .AllFailed, for example:

```
depends: "task-1.AnySucceeded || task-2.AllFailed"
```
   
## Compatibility with `dependencies` and `dag.task.continueOn`

This feature is fully compatible with `dependencies` and conversion is easy.

To convert simply join your `dependencies` with `&&`:

```
dependencies: ["A", "B", "C"]
```

is equivalent to:

```
depends: "A && B && C"
```

Because of the added control found in `depends`, the `dag.task.continueOn` is not available when using it. Furthermore,
it is not possible to use both `dependencies` and `depends` in the same task group.
