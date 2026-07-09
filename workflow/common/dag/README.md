# DAG Evaluation Package

This package implements DAG dependency evaluation for Argo Workflows. It determines which tasks are ready to execute, which are waiting on dependencies, and which should be omitted because their depends conditions can never be satisfied.

Both DAG and Steps template types use this package via the `Engine` in `workflow/controller/engine.go`.

## Files

| File | Purpose |
|------|---------|
| `argo.go` | `DAGEvaluator` — main evaluator with readiness checking, cascading omission, and public API |
| `topology.go` | `WorkflowTasks` — task collection, dependency resolution, depends expression parsing, topology caching |
| `store.go` | `WorkflowStore` — maps task names to workflow nodes, tracks evaluator-managed state (e.g. Omitted) |
| `task.go` | `Task` interface and `DAGTask` adapter for `wfv1.DAGTask` |
| `types.go` | Shared types: `TaskState`, `ReadinessResult`, `TaskResult`, `EvaluationResult` |
| `expansion.go` | WithItems/WithParam/WithSequence task expansion |

## How It Works

### 1. Construction

```go
evaluator := dag.NewDAGEvaluator(wf, tmpl, boundaryID, boundaryName)
```

This creates:
- A `WorkflowStore` that reads node state from `wf.Status.Nodes`
- A `WorkflowTasks` that parses depends expressions and caches the dependency topology

### 2. Topology Caching

`WorkflowTasks` pre-computes the dependency graph (which tasks depend on which, and the normalized depends expressions) at construction time. Since a new `DAGEvaluator` is created each reconciliation cycle, the topology is recomputed each time.

Task names are hex-encoded (e.g., `my-task` → `t6d792d7461736b`) so they're safe identifiers in expression evaluation.

### 3. Readiness Evaluation

For each task, `evaluateDependsReadiness` builds an evaluation scope from dependency node states and evaluates the depends expression via `argoexpr.EvalBool()`:

- **Ready**: Expression is true (or no depends expression and no pending deps)
- **Waiting**: Expression is false but some deps are still pending — could become true later
- **Omit**: Expression is false and either all deps are terminal, or even the best-case outcomes for pending deps can't satisfy it

The "best-case" check is key: if dep A is Omitted and the expression requires `A.Succeeded`, we try setting all pending deps to all-true. If it's still false, the expression is structurally unsatisfiable → Omit.

### 4. Cascading Omission

`evaluateAllStates` runs a fixed-point loop:
1. Clear previously-omitted states (conditions may have changed since last call)
2. Evaluate all pending tasks
3. If any are newly Omitted, loop again (downstream tasks may now be unreachable)
4. Stop when no changes occur

This handles chains like: A fails → B (depends on A.Succeeded) is Omitted → C (depends on B) is Omitted.

### 5. Public API

The `Engine` in `engine.go` uses these methods:

```go
evaluator.GetTargetTasks(ctx)      // Leaf tasks or explicit DAG targets
evaluator.EvaluateAll(ctx)         // Map of task → EvaluationResult
evaluator.EvaluateTask(ctx, name)  // Evaluation result for a single task
evaluator.GetDependencies(ctx, t)  // Dependencies for a specific task
```

Each `EvaluationResult` contains:
- `ShouldRun` — task is ready to execute
- `Suspended` / `WaitingOn` — task is waiting for specific dependencies
- `Skipped` / `SkipReason` — task will never run (depends condition unsatisfiable)

## Architecture

```
Engine (engine.go)
  │
  ├── DAGEvaluator (argo.go)
  │     │
  │     ├── WorkflowTasks (topology.go)
  │     │     ├── Dependency graph (pre-computed)
  │     │     ├── Depends expression parsing
  │     │     └── Task name normalization
  │     │
  │     └── WorkflowStore (store.go)
  │           ├── Node lookup by task name
  │           ├── State tracking (Omitted, etc.)
  │           └── Hooks fulfillment checking
  │
  └── Task interface (task.go)
        ├── DAGTask (for DAG templates)
        └── StepAdapter (in steps.go, for Steps templates)
```
