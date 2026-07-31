Description: Fix daemon nodes incorrectly displayed as Failed after controlled shutdown
Authors: [Jeremiah Trest](https://github.com/JerT33)
Component: Controller
Issues: 14790, 2762

Daemon nodes were incorrectly marked NodeFailed by `assessNodeStatus` when their pod exited after a controlled shutdown (SIGTERM via `killDaemonedChildren`). This caused daemon nodes to display as red/Failed in the UI even when the workflow completed successfully, and could trigger spurious retries when `retryStrategy` was set, eventually failing the entire workflow.

The fix checks whether `killDaemonedChildren` already cleared the `Daemoned` flag (nil = controlled shutdown) before applying NodeFailed, preserving the NodeSucceeded phase that `killDaemonedChildren` set.
