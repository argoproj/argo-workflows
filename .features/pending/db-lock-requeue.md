Description: Requeue workflows instead of failing them when database lock retries are exhausted
Authors: [Isitha Subasinghe](https://github.com/isubasinghe)
Component: General
Issues: 16101

Database semaphores and mutexes acquire their locks in a `SERIALIZABLE` transaction, which the database may abort under contention.
The controller retries those aborts with a short backoff, but a workflow that exhausted the retries was failed outright.
Exhausting the retries on a driver-reported transaction conflict is now treated the same as not getting the lock: the workflow reports `Waiting for lock: database contention, will retry` and is requeued to try again.
For workflow-level locks the workflow stays pending; for template-level locks the workflow stays running with the node waiting on the lock.
Any other database error still fails the workflow as before.
