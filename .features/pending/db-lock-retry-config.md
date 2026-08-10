Description: Configurable retry backoff for database lock serialization failures, and requeue instead of failing the workflow
Authors: [Isitha Subasinghe](https://github.com/isubasinghe)
Component: General
Issues: 16101

Database semaphores and mutexes acquire their locks in a `SERIALIZABLE` transaction, which the database may abort under contention.
The controller retries those aborts, but the backoff was hardcoded, and a workflow that exhausted the retries was failed outright.
The backoff is now configurable under `synchronization.dbRetryConfig`, which takes `steps`, `duration`, `factor`, `jitter` and `cap`.
Every field left unset keeps its previous default, so `steps` can be raised on its own if your database reports frequent serialization failures at scale.
Keep `cap` low: the controller holds its sync lock for the whole retry loop, so long sleeps stall lock acquisition for every other workflow.
Exhausting the retries now leaves the workflow pending and requeues it after 10 seconds, rather than failing it.
Set `requeue: false` to restore the previous behavior.
See [the workflow controller configmap](workflow-controller-configmap.md#dbretryconfig).
