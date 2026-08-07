Description: Archive the `init` and `wait` container logs, not just the `main` container
Authors: [myzk-a](https://github.com/myzk-a)
Component: General
Issues: 12640

Set `archiveSystemContainerLogs: true` to also archive the logs of Argo's own system containers — the `init` container, which loads input artifacts, and the `wait` container, which saves outputs and logs.
By default only the `main` container's logs are archived.

The system container logs are stored alongside `main-logs` as artifacts named `init-logs` and `wait-logs`, and can be viewed from the Argo UI for garbage-collected Pods.
Under the opt-in init-less pod layout, the single `supervisor` container performs both roles, so its log is archived as one `supervisor-logs` artifact instead.
This is useful for inspecting what the system containers did after the Pod is gone, such as debugging output or artifact upload failures in `wait`, or reviewing which input artifacts the `init` container loaded.

`archiveSystemContainerLogs` is controlled separately from `archiveLogs`, so you only pay to store these extra logs when you need them.
It can be set at the workflow-controller config-map, workflow spec, or template level, following the same priorities as `archiveLogs`.

In the legacy pod layout a *failing* `init` container is not captured, because the `wait` container never starts to archive it; the init-less layout does capture the `supervisor` log even when loading input artifacts fails.
See [Configuring Archive Logs](https://argo-workflows.readthedocs.io/en/latest/configure-archive-logs/) for the example manifest and the full list of limitations.
