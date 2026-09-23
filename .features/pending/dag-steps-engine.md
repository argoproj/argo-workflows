Description: DAG and Steps templates run on one engine, and `depends` takes effect as soon as it is settled
Authors: [Isitha Subasinghe](https://github.com/isubasinghe)
Component: General
Issues: 12192 16450 16454

DAG and Steps templates are now executed by a single engine, so fixes to readiness, retries, hooks and omission apply to both template types.

A DAG task's `depends` expression is re-evaluated whenever a task it references changes state, and takes effect as soon as its result is settled rather than only once every referenced task has finished.
For example `depends: "task-1 || task-2"` runs as soon as either bare reference is satisfied, and `depends: "task-1.Succeeded && task-2"` is omitted as soon as `task-1` fails.
Expressions whose result still depends on a pending task wait as before.
See [When a task runs](enhanced-depends-logic.md#when-a-task-runs) and the [upgrade note](upgrading.md#dag-depends-expressions-take-effect-as-soon-as-they-are-settled), which shows how to keep the old wait-for-all behaviour.

A step group whose every step was omitted is now `Omitted` rather than `Succeeded`, so that retrying or resubmitting a workflow that failed in an earlier group works.
