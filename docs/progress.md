# Workflow Progress

![alpha](assets/alpha.svg)

> v2.12 and after

When you run a workflow, the controller will report on its progress.

We define progress as two numbers, `N/M` such that `0 <= N <= M and 0 <= M <= 1`. 

* `N` is the number of completed tasks.
* `M` is the total number of tasks.

E.g. `0/0`, `0/1` or `50/100`.

Unlike [estimated duration](estimated-duration.md), progress is deterministic. I.e. it will be the same for each workflow, regardless of any problems. 

Progress for each node is calculated as follows:

2. For a pod node either `1/1` if completed or `0/1` otherwise.
3. For non-leaf nodes, the sum of its children.

For a whole workflow's, progress is the sum of all its leaf nodes.
 
!!! Warning 
    `M` will increase during workflow run each time a node is added to the graph.