// Package poolpod is a pool pod's side of hot workers: the file protocol
// between its `worker` and `main` containers, and what `main` needs to run
// one task after another in the same container without them seeing each
// other: the process sweep and the isolation of each task's files.
//
// The protocol works over the shared /var/run/argo volume. The pool-level
// directory holds `current`, the path of the task directory `main`
// should run next, and `main`, main's liveness lock, held exclusively for
// as long as main is taking tasks, which the kernel releases however main
// dies: the worker claims a task only while main holds it, so a main that
// can't start (its image can't be pulled, or its env names a missing
// Secret) leaves the pool's tasks queued. Each task directory holds:
//
//   - `task`: the task's TaskSpec, written by the worker before it offers
//     the task.
//   - `ready`: written by the worker once the task directory is complete,
//     just before the worker points `current` at it.
//   - `taken`: created exclusively, so whoever creates it first wins: by
//     main with the content `main` when it starts the task, or by the worker
//     with `withdrawn` to stop main ever starting it.
//   - `lock`: main's liveness lock, held exclusively while main has the task,
//     which the kernel releases however main dies.
//   - `message`: optional, main's message for a task it failed itself (such
//     as an input path that exists in the image), written before `exitcode`.
//   - `exitcode`: the task's exit code, written by main before it releases
//     `lock`.
//   - `stop`: written by the worker to make main stop the task, holding the
//     reason.
//
// Where task directories live is not this package's business: the
// executor's per-task path layout decides (in a pool pod, a directory per
// task in the pool directory), and both sides name a task by its directory.
package poolpod
