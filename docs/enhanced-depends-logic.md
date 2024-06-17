# Enhanced Depends Logic

> v2.9 and after

Enhanced `depends` improves on [the `dependencies` field](walk-through/dag.md) by specifying which _result_ of a task to depend on.
For example, to only run a task if its dependent task succeeded.

## Depends

You can use the `depends` field to specify dependent tasks, their results, and boolean logic between them.

You use operands of the form `<task-name>.<task-result>`, such as `task-1.Succeeded`, `task-2.Failed`, `task-3.Daemoned`.
Available task results are:

|  Task Result | Description |
|:------------:|-------------|
| `.Succeeded` | Task finished with no error |
| `.Failed`    | Task exited with a non-0 exit code |
| `.Errored`   | Task had an error other than a non-0 exit code |
| `.Skipped`   | Task's [`when`](walk-through/conditionals.md) condition evaluated to `false` |
| `.Omitted`   | Task's `depends` condition evaluated to `false` |
| `.Daemoned`  | Task is [daemoned](walk-through/daemon-containers.md) and is not `Pending` |

For compatibility with `dependencies`, an unspecified result is equivalent to `(task.Succeeded || task.Skipped || task.Daemoned)`. For example:

```yaml
depends: "task || task-2.Failed"
```

is equivalent to:

```yaml
depends: (task.Succeeded || task.Skipped || task.Daemoned) || task-2.Failed
```

You can use boolean logic with the operators:

* `&&`
* `||`
* `!`

Example:

```yaml
depends: "(task-2.Succeeded || task-2.Skipped) && !task-3.Failed"
```

If you depend on a task that uses `withItems`, you can use `.AnySucceeded` and `.AllFailed`. For example:

```yaml
depends: "task-1.AnySucceeded || task-2.AllFailed"
```

## Compatibility with `dependencies` and `dag.task.continueOn`

You cannot use both `dependencies` and `depends` in the same task group.

To convert from `dependencies` to `depends`, join your array into a string with `&&`. For example:

```yaml
dependencies: ["A", "B", "C"]
```

is equivalent to:

```yaml
depends: "A && B && C"
```

`dag.task.continueOn` is not available when using `depends`; instead you can specify `.Failed`.
