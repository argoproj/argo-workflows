Description: DAG and Steps templates run on one engine
Authors: [Isitha Subasinghe](https://github.com/isubasinghe)
Component: General
Issues: 12192 16450 16454

DAG and Steps templates are now executed by a single engine, so fixes to readiness, retries, hooks and omission apply to both template types.

A step group whose every step was omitted is now `Omitted` rather than `Succeeded`, so that retrying or resubmitting a workflow that failed in an earlier group works.
