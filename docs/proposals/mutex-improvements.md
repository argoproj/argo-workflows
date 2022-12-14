# Proposal for Mutex/Semaphore improvements

## Introduction

The motivation for this is to improve the reliability of how locking works via mutexes and semaphores. Currently the implementation makes use of
string formatting, this is not scalable (with respect to the size of developers and features).

### Why is this needed?

### How do mutexes currently work?

Nearly all of the code regarding how mutexes work reside in `sync_manager.go`.
Here is an example run of how locks are acquired and released. Some parts have been omitted for brevity, I recommend opening up the file and following through the
examples below.

`
getHolderKey({"Namespace": "argo", Name: "example"}, "node") = "argo/example/node"

-- MutexStatus After LockAcquired Call --
items = ["argo", "example", "node"]
holdingName = "node"
ms.Holding = [MutexHolding{Mutex: lockKey, Holder: holdingName}]

getResourceKey("argo", "example", "node") = "argo/example/node"
`

This works fine but let's examine another case where this breaks. This is the bug from issue <https://github.com/argoproj/argo-workflows/issues/8684>

`
getHolderKey({"Namespace": "argo", Name: "deadlock-test-sn8p5"}, "deadlock-test-sn8p5") = "argo/deadlock-test-sn8p5/deadlock-test-sn8p5"

-- "MutexStatus" After LockAcquired Call --
items = ["argo", "deadlock-test-sn8p5", "deadlock-test-sn8p5"]
holdingName = "deadlock-test-sn8p5"
ms.Holding = [MutexHolding{Mutex: lockKey, Holder: holdingName}]

getResourceKey("argo", "deadlock-test-sn8p5", "deadlock-test-sn8p5") = "argo/deadlock-test-sn8p5"
`

### A criticism of the current approach

The current approach is heavily dependent on string formatting. This makes it incredibly difficult to understand functionality when there are no comments or documentation
outlining the logic used. On top of this, it makes the "Find References" functionality on LSP powered editors near useless. My understanding of the function `getResourceKey` is that
it seems to construct the holder key that is generated via `getHolderKey` through using processed strings generated via `LockAcquired`.
I strongly believe this needs to be refactored to be less dependent on string formatting. I strongly believe that the key obtaining process should be far more obvious, the disconnect between key acquisition and release is inherently flawed and will only lead to further bugs down the line.

### How should they work?

We should not be generating information after it has already been generated. It is extremely important to only maintain a single source of truth,
generating this information (the holder key) produces sources of truth at every generation point. Ideally we should be storing this information somewhere
as we will be needing it eventually when a release call is made.

### Solutions

#### Store the holder key directly in the MutexStatus/SemaphoreStatus structs

I may be wrong here but I don't see why it wouldn't be possible to store this information directly in the status structs.
This seems to be the most simple way of ensuring that there is only a single source of truth. I didn't go with this solution in my existing PR because this would require changing the information MutexStatus/SemaphoreStatus held. But given this
discussion is being opened in a proposal, it seems plausible to go with this solution instead.

##### Advantages of solution #1

* More sensible approach than including a new dependency (ConfigMap).
* Relatively simple change.
* Single source of truth

##### Disadvantages of solution #1

* Changing a field could be major change and might break projects in the ecosystem that rely on this behavior. This has been the behavior of mutexes ever since they have been implemented, it is high risk to change this in my opinion.

#### Store holding/pending information in a config map

We can store the holder keys inside a config map, on release, we refer to the holder keys inside this config map. There is an existing WIP PR for this, [here](https://github.com/argoproj/argo-workflows/pull/10009).
It requires handling pending workflows. If we are going with this solution, a small amendment to deal with pending items will have to be made, we may have to introduce two config maps. One will be used for storing information regarding acquired locks, the other for storing information regarding pending lock acquisitions.
There is a possibility that it might be possible to use a single config map here, but that solution needs to be explored in order to confirm this.

##### Advantages of solution #2

* The change in behavior is transparent to users.
* Existing PR should allow for us to be quite quick in pushing a fix.

##### Disadvantages of solution #2

* Fairly large change.
* Introduces error paths, that being said these error paths being encountered is a bigger problem than the mutex issue itself. The error path here is encountered if the Kubernetes API is down.
* Two sources of truth, the data structure and also the ConfigMap used to store metadata.

#### Recommendation

My recommendation is dependent on whether we classify the first solution as a major/breaking change or not. If it is a breaking change the second approach is clearly better as it is transparent
to the users. I think in this case, it is potentially better to push the second solution as a minor change as a temporary fix and introduce the first when we publish the next major version.

**Even if we don't go with either of these approaches, I strongly believe we should at least change how the mutex acquisition and release mechanism works at the moment because currently it is inherently flawed.**

## Resources

* [Open PR](https://github.com/argoproj/argo-workflows/pull/10009)
* [Issue that prompted this discussion](https://github.com/argoproj/argo-workflows/issues/8684)
