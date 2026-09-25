Description: Opt-in pod layout that removes the `argoexec init` container
Authors: [Alan Clucas](https://github.com/Joibel)
Component: General
Issues: 16154

New controller-wide `initlessPod` mode (workflow controller ConfigMap) that eliminates the `argoexec init` container. Beta: off by default and may change in incompatible ways in future minor releases before being promoted to stable.
The `argoexec` binary is mounted into `main` via a Kubernetes image volume (KEP-4639 — Beta in K8s 1.33 behind a feature gate, GA in 1.36), and a new `supervisor` container handles template write, script staging, input artifact download, readiness signaling, and the post-main responsibilities previously held by `wait`.
Artifact plugins run as regular sidecars invoked by `supervisor` for both Load and Save instead of as init containers, so pods run with zero init containers.
Off by default; `wait` and the legacy pod layout remain unchanged.
Enable by setting `initlessPod.enabled: true` in the workflow controller ConfigMap — every subsequently scheduled workflow pod uses the init-less layout.
Rollback by setting it back to `false`; in-flight pods keep their original layout.
