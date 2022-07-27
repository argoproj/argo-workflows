# Resource Sharing

It could be the case that a particular template execution uses 1 of a finite number of resources. Examples of this could be software licenses or compute nodes, just to name two examples.

## Examples

### Example 1

At a particular moment, we have two users, Alice and Bob, who are trying to render images via Workflows that we have submitted on their behalf. The software running inside the render step’s Template requires access to a license, and licenses are scarce resources. Therefore, at this particular moment, we give Alice and Bob each exactly half of the available licenses. (Executing a template will cause exactly one license to be claimed.)

### Example 2

Somebody is running an in-house cluster with only a few nodes of a particular type. A particular Template’s Pod has a toleration towards these nodes, but there are enough concurrent Pods that just a small number of them will end up blocking other users’ workflows from progressing.

### Example 3

??

## Proposal

Please see related issue [here](https://github.com/argoproj/argo-workflows/issues/8982).

## Discussion

`getResourceAllowance` is the core of this feature. It's called every time a new Workflow is
popped off of the workflow queue, and the number of resources that are available _for this
resource sharing ID_ are calculated. Then, if `operate` tries to execute more than this
number of template instances, we return `ErrParallelismReached` and try again later.

### How is this different than semaphores?

The main difference is that semaphores would achieve parallelism, but would not distribute resources evenly across “resource sharing IDs”. So it could be the case that 100 Workflows are submitted on behalf of user A, and then B submits a single workflow but then has to wait for all/many of A’s Workflows to complete.

## New concepts

`ResourceSharingID`: This is the "user" that we are submitting this Workflow on behalf of. I
decided to not use the word "user" to avoid ambiguity.

`Resource`: A resource is directly related to templates. Some templates use resources. When
one of these template are executed, one resource is being used.
