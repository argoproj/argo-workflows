Description: Run container and script templates on pools of long-lived worker pods with `pool:` (experimental)
Authors: [Alan Clucas](https://github.com/Joibel)
Component: General
Issues: 16530

Hot workers run container and script templates on a pool of long-lived worker pods, instead of creating a pod for each attempt of a step or DAG task.
Each worker pod runs one attempt after another, which suits many short steps or DAG tasks.
Add `pool:` to a container or script template, for example `pool: {name: python-squares, initialReplicas: 2, retentionSeconds: 900}`, or to `templateDefaults`.
Templates with the same pool name and pod settings share a pool, and the controller creates each pool when its first attempt is ready to run and deletes it once it has been idle for its retention.
This feature is experimental and off by default.
To enable it, add the `manifests/components/hot-workers` RBAC component to your install, set `hotWorkers.enabled: true` in the `workflow-controller-configmap`, and restart the controller.
The account the worker pods' `worker` container runs as needs the verbs of the quick-start's `hot-worker` Role: the pool's service account, or the executor's if you set `executor.serviceAccountName`.
The attempts a pool runs share its worker pods' files and credentials, and some template fields, such as `sidecars` and `podSpecPatch`, are rejected on pooled templates.
See the hot workers page of the documentation for how attempts on a pool differ from pods.
