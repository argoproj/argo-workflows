Description: DAG and Steps templates run on one engine
Authors: [Isitha Subasinghe](https://github.com/isubasinghe)
Component: General
Issues: 14767

DAG and Steps templates are now executed by a single engine, so fixes to readiness, retries, hooks and omission apply to both template types.
Most workflows behave as before.
The "DAG and Steps templates run on one engine" section of the upgrade notes (docs/upgrading.md) describes every change, including:

  - A step group that follows a failed, stopped or timed-out group is now shown as `Omitted`, together with its steps; previously it did not appear.
  - An expanded step now has a `TaskGroup` node holding its items, as an expanded DAG task has.
  - Behavior also changes for errors and `continueOn`, lifecycle and exit hooks, step groups, `globalName` outputs, metrics and `argo retry`.
  - Running Steps workflows with an expanded step do not survive a rollback of the controller; the upgrade notes say what to do before rolling back.
