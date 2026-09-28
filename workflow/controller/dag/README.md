# DAG Evaluation Package

This package decides which tasks of a DAG or Steps template are ready to run, which are waiting on dependencies, which should be omitted because their `depends` condition can never be satisfied, and what a retry or task-group node's current state amounts to.
It performs no side effects: the `Engine` in `workflow/controller/engine.go` reads its results and creates, dispatches and marks nodes.
Both template types use it — Steps tasks are adapted to the same `Task` interface with synthetic dependencies on the previous step group.

## Files

| File | Purpose |
|------|---------|
| `doc.go` | Package comment |
| `argo.go` | `DAGEvaluator` — readiness evaluation, cascading omission, retry and task-group assessment, public API |
| `topology.go` | `WorkflowTasks` — task collection, dependency resolution, topological order |
| `store.go` | `workflowStore` — maps task names to workflow nodes; `TaskNodeName` / `TaskNameFromNodeName` naming convention |
| `task.go` | `Task` interface and the `DAGTask` adapter for `wfv1.DAGTask` (`StepAdapter` lives in `workflow/controller/steps.go`) |
| `types.go` | `EvaluationResult`, `Action`, and the `taskResult` scope struct |
| `expansion.go` | `withItems` / `withParam` / `withSequence` expansion and expanded task naming |
| `helpers_test.go` | Test-only conveniences (`NewDAGEvaluator`, `EvaluateTask`); production code does not use them |

## How it works

### 1. Construction

```go
evaluator := dag.NewDAGEvaluatorFromTasks(wf, tasks, tmpl, boundaryID, boundaryName)
evaluator.SetRetryStrategy(taskName, retryStrategy) // per task with a retry strategy
evaluator.SetRetryDecider(taskName, decider)        // the engine's retry policy/expression decision
```

`tasks` is the boundary's task list as `dag.Task` values (`DAGTask` for DAG templates, `StepAdapter` for Steps).
Construction builds a `workflowStore` over `wf.Status.Nodes` and a `WorkflowTasks` that resolves every task's dependencies once.
A new evaluator is created every reconcile cycle, so nothing here is long-lived.

### 2. Dependency resolution

A user-written `depends` expression is tokenized with `common.ParseDepends` — the same grammar workflow validation uses, so an expression cannot pass validation and be read differently here.
Legacy `dependencies` lists (and the synthetic dependencies of Steps tasks, whose names are `[<group index>].<step name>`, for example `[0].build`) are structured data and are expanded directly with `common.ExpandDependency`; they are never re-parsed as an expression.

Task names are rewritten to hex-encoded identifiers (`my-task` → `t6d792d7461736b`) so they are valid, collision-free identifiers in the evaluated expression.
The dependency graph is sorted topologically (Kahn's algorithm) once, at construction.

### 3. Readiness evaluation

A task **waits** while any dependency is still pending: not started, running, a retry node whose outcome the engine has not recorded yet, or finished with lifecycle or exit hooks still running.
A running daemon counts as finished.
Once every dependency has finished, `evaluateDependsReadiness` builds a scope of dependency states — a `taskResult` per dependency with the fields `Succeeded`, `Failed`, `Errored`, `Skipped`, `Omitted`, `Daemoned`, `AnySucceeded`, `AllFailed` (the same vocabulary as `common.TaskResult*`) — and evaluates the normalized expression with a cached, compiled `expr` program.
The task is **ready** if the expression is true and is **omitted** if it is false.

### 4. Cascading omission

`evaluateAllStates` clears any previously computed Omitted state and evaluates every task in topological order in a single pass.
Because a task is evaluated after all of its dependencies, an omission propagates in the same pass: A fails → B (`depends: A.Succeeded`) is omitted → C (`depends: B`) is omitted.

### 5. Retry and task-group nodes

`evaluateRetryNode` is the pure assessment counterpart of the controller's `processNodeRetries`: it inspects a retry node's attempts and reports whether to execute another attempt, wait (`RequeueAfter`, computed with `common.RetryBackoffWait` minus the time already elapsed), succeed, or fail.
Whether the policy allows another attempt is decided by the engine-provided `RetryDecider`, so the evaluator and the operator cannot disagree.
Retry strategies are registered under static task names; expanded children (`A(0:x)`) inherit their task's strategy.

`EvaluateAll` also emits a result per expanded child of a task group (`ParentTaskName` set), and `evaluateTaskGroupNode` derives the group's phase from its children.

### 6. Public API

What the `Engine` uses:

```go
evaluator.EvaluateAll(ctx)              // map of task name → EvaluationResult (incl. expanded children)
evaluator.GetTargetTasks(ctx)           // explicit dag.target tasks, or the leaves
evaluator.FindLeafTaskNames(ctx)        // tasks nothing depends on
evaluator.GetAncestors(ctx, task)       // transitive dependencies (unordered)
evaluator.GetDependencies(ctx, task)    // direct dependencies
evaluator.GetTask(name)                 // the Task by name
dag.ExpandTask / dag.HasExpansion       // withItems/withParam/withSequence expansion
dag.TaskNodeName / dag.TaskNameFromNodeName // task ↔ node name convention
```

Fields of `EvaluationResult` the engine acts on:

- `Action` — the evaluator's decision for a retry or task-group node (`ActionExecute`, `ActionSucceed`, `ActionFail`, `ActionNone`); `ShouldRun` — the task's dependencies allow it to run.
  The engine dispatches the task for Execute, Succeed and Fail alike: for Succeed and Fail the operator's retry handling or the TaskGroup assessment records the outcome.
- `CurrentPhase` and `FulfilledForDeps` — for boundary phase assessment and dependency gating (a running daemon is fulfilled for its dependants).
- `RequeueAfter` — retry backoff still to wait.
- `Skipped` / `SkipReason` — the task will never run; the engine creates the Omitted node with this reason.
- `Error` — the task could not be assessed; the engine records a terminal Error node.
- `ParentTaskName` — set on expanded-child results so the engine can dispatch them without parsing the name.

`ActionReason`, `Suspended` and `WaitingOn` are diagnostic: the engine logs them at debug level and does not act on them.

## Architecture

```text
Engine (workflow/controller/engine.go)
  │
  ├── DAGEvaluator (argo.go)
  │     ├── WorkflowTasks (topology.go)
  │     │     ├── common.ParseDepends / ExpandDependency (shared with validation)
  │     │     ├── hex-encoded identifiers
  │     │     └── topological order
  │     ├── workflowStore (store.go)
  │     │     ├── node lookup by task name (TaskNodeName)
  │     │     ├── evaluator-managed state (Omitted)
  │     │     └── hook-fulfilment checks
  │     └── RetryDecider / RetryStrategy registered by the engine
  │
  └── Task interface (task.go)
        ├── DAGTask   (DAG templates)
        └── StepAdapter (Steps templates, workflow/controller/steps.go)
```
