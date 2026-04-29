Issues: 16054
Authors: [Anton Pechenin](ntny1986@gmail.com)
Description: Add batching support for nodeResults in TaskSet patch operations
Component: General

You can now batch nodeResults when patching Workflow TaskSets.
This improves reliability for workflows with a large number of parallel nodes by preventing patch payloads from exceeding etcd size limits.

Previously, workflows with many parallel nodes (tested with ~200), particularly when using the agent executor plugin, could generate patch requests that exceeded etcd size limits when multiple TaskSet updates occurred within a single reconciliation period.
With batching enabled, nodeResults are split into smaller chunks and patched sequentially, keeping each request within safe size limits.

For example, instead of sending a single large patch, the agent will split results into batches (e.g., 30 nodes per request), ensuring each patch stays within safe size limits.