# Init-less Pod Layout

> v4.1 and after

!!! note "Beta"
    Init-less pod mode is a beta feature. It is off by default and may change in incompatible ways in future minor releases before being promoted to stable. The legacy pod layout (init + wait) remains the default and is unchanged.

Init-less pod mode is an **opt-in** controller-wide pod layout that eliminates the `argoexec init` container. It relies on Kubernetes [image volumes](https://kep.k8s.io/4639), which are Beta (feature-gate opt-in) from Kubernetes 1.33 and enabled by default from 1.36 onwards.

## Why

Every workflow pod scheduled by a controller in legacy mode has:

- `init` — an init container that copies the `argoexec` binary onto a shared `emptyDir`, writes the template JSON, stages script source, and downloads non-plugin input artifacts.
- Per-plugin `init-artifact-<plugin>` init containers that each run their plugin's server just long enough to download one set of input artifacts.
- `wait` — a regular container that observes `main`, captures outputs, and saves artifacts and logs post-main.
- `main` — the user's container, with `argoexec emissary` injected as its PID 1.
- Per-plugin `artifact-plugin-<plugin>` sidecars that handle the output-artifact Save path.

Init containers run sequentially before any regular container, adding startup latency to every pod. The init container's responsibilities also overlap conceptually with `wait` — both are Argo-infrastructure containers running `argoexec`. Plugin authors must implement both an init-mode invocation (Load) and a long-running-mode invocation (Save), and a plugin used for both inputs and outputs ends up as two containers per pod.

## What init-less mode does

| | Legacy | Init-less |
| --- | --- | --- |
| Init containers | `init` + one per input plugin | **zero** |
| Auxiliary container | `wait` | `supervisor` |
| argoexec binary delivery | copied by `init` onto shared emptyDir | mounted into `main` via an image volume |
| Plugin containers | init (Load) + sidecar (Save); up to 2 per plugin | one sidecar per unique plugin, driven by `supervisor` for both Load and Save |

The net effect is zero init containers per pod, no sequential init phase, a single plugin invocation path, and a simpler mental model.

## Enabling

Edit the workflow controller ConfigMap (usually `workflow-controller-configmap`):

```yaml
initlessPod: |
  enabled: true
```

After saving the ConfigMap, newly scheduled workflow pods use the init-less layout. In-flight pods keep whatever layout they were scheduled with — there is no migration.

This is a **controller-wide** setting with no per-workflow override.

## Rollback

Set `enabled: false` (or remove the field). Newly scheduled pods revert to the legacy layout. In-flight pods keep their original layout.

## Requirements

- **Kubernetes with image volumes available** on every node that runs workflow pods. Image volumes (KEP-4639) are Beta in 1.33–1.35, meaning the `ImageVolume` feature gate must be enabled on both the API server and kubelet on those versions. From 1.36 onwards the feature is GA and enabled by default. On earlier versions, or on 1.33–1.35 without the feature gate enabled, pod creation fails with a standard Kubernetes API error ("unknown field" or "feature gate disabled"). The controller does not probe the cluster — operators own their version and feature-gate configuration.
- **Agent pods are unchanged** — init-less mode applies to workflow pods only. HTTP / resource templates continue to use agent pods with their existing init container.
- **Image pull secrets** on the pod apply to image-volume sources. If your controller uses a private registry for the executor image, ensure the workflow's service account has image pull secrets configured as usual.
- **Multi-arch** — the image volume must match the node's architecture. If you build a custom `argoexec` image, ensure its manifest list covers every architecture your nodes run.

## How it works

### Container layout

```yaml
pod:
  volumes:
    - name: argoexec-bin                # NEW in init-less mode
      image:
        reference: quay.io/argoproj/argoexec:<tag>   # from executor.image
        pullPolicy: <executor.imagePullPolicy>       # reuses the executor pull policy, not a fixed value
    - name: var-run-argo
      emptyDir: {}

  # NO init containers

  containers:
    - name: supervisor
      image: quay.io/argoproj/argoexec:<tag>
      command: [argoexec, supervisor]
      # /var/run/argo plus /tmp, the input-artifacts volume (when the template
      # has input artifacts) and each plugin's socket volume (when it uses
      # artifact plugins) — abbreviated here.
      volumeMounts: [ /var/run/argo, ... ]
      # ARGO_TEMPLATE, ARGO_ARTIFACT_PLUGIN_NAMES, ARGO_INPUT_ARTIFACT_PLUGIN_NAMES, ...

    - name: main
      image: <user image>
      command: [/argo-bin/bin/argoexec, emissary, --, <user command...>]
      volumeMounts:
        - { name: argoexec-bin, mountPath: /argo-bin, readOnly: true }
        - { name: var-run-argo, mountPath: /var/run/argo }
      env:
        - { name: ARGO_WAIT_FOR_READY, value: "true" }

    - name: artifact-plugin-<name>      # one per unique plugin used by inputs or outputs
      image: <plugin image>
      ...
```

### Startup sequence

1. Pod scheduled. The image volume is mounted before containers start.
2. `supervisor` and `main` start concurrently (no K8s ordering guarantee).
3. `main`'s emissary blocks on the `/var/run/argo/status` marker before reading the template (gated by the `ARGO_WAIT_FOR_READY=true` env var). The marker's first line is a state token — `RUNNING`, `READY`, or `FAILED` (with the failure message on following lines).
4. `supervisor` in order:
    1. Writes the status marker as `RUNNING` and starts a heartbeat goroutine that rewrites it every 5s. Each rewrite advances the marker's mtime; `main` treats a marker that neither appears nor advances within 30s as a dead supervisor and fails fast (rather than hanging to the pod deadline). The initial write also overwrites any stale marker left by a prior attempt.
    2. Writes `/var/run/argo/template`.
    3. Calls `StageFiles` (for script templates).
    4. In parallel (errgroup): loads non-plugin input artifacts, and for each input-plugin name invokes Load on the plugin's gRPC socket.
    5. Stops the heartbeat, then on success: atomically writes `READY`.
    6. On failure: atomically writes `FAILED` plus the error text, then falls through to PostMain (waits for `main` to exit and captures its logs). Emissary in `main` reads the `FAILED` status, exits with a distinct code (65) so the controller can attribute the failure to supervisor pre-main.
5. Emissary sees `READY`, reads the template, symlinks each input artifact into its expected path (see [Input artifacts: symlink vs bind mount](#input-artifacts-symlink-vs-bind-mount)), then execs the user command.
6. `supervisor` continues with the post-main responsibilities: observes `main`, captures outputs, saves output artifacts (plugin and non-plugin), saves logs, reports outputs. For plugin-backed output artifacts it invokes Save on the same sidecar it already used for Load.
7. Once Save completes, `supervisor` signals plugin sidecars to exit; the pod terminates.

### Failure handling

| Scenario | Behavior |
| --- | --- |
| Artifact download fails in `supervisor` pre-main | `supervisor` writes `FAILED` plus the error to `/var/run/argo/status`, then continues to PostMain (waits for `main`, captures logs). Emissary in `main` reads the `FAILED` status, logs, exits with code 65. Pod fails; controller marks the node `Error`. |
| `supervisor` dies before writing a terminal status | The status marker stops being heartbeated. `main`'s emissary sees its mtime go stale (no update within 30s — or the marker never appearing at all) and fails fast with a "supervisor presumed dead" error, rather than hanging to the pod's `activeDeadlineSeconds`. (The pod uses `restartPolicy: Never`, so the supervisor is **not** restarted, the same as the legacy `wait`/init containers.) |
| Plugin container fails to start | `supervisor` times out reaching the plugin's socket (120s). Writes `FAILED` plus the error, then continues to PostMain (logs only — no outputs to save). |
| Image volume pull fails | Kubernetes surfaces `ImagePullBackOff` on `main` — same user experience as any other container image failure. |
| User image is distroless/scratch | Works — `argoexec emissary` from `/argo-bin` is the entrypoint; init-less mode does not depend on anything in the user image. |

## Input artifacts: symlink vs bind mount

In legacy mode, each input artifact is delivered to `main` via a per-artifact Kubernetes `SubPath` mount — kubelet bind-mounts the downloaded file onto the artifact's `path` inside the container. The result is a **regular file** at that path.

In init-less mode, `main` and `supervisor` start concurrently. If kubelet sets up a `SubPath` mount before supervisor has written the file, it pre-creates the path as an empty directory in the shared emptyDir, which then causes supervisor's artifact-rename to fail. To avoid this race, init-less mode:

- Mounts the whole `input-artifacts` emptyDir read-write on `main` at `/argo/inputs/artifacts` (no `SubPath`).
- Has the emissary, immediately after the status marker fires and before executing the user command, create a **symlink** at each input artifact's path pointing to the underlying file in the shared emptyDir.

**What this means for workflow authors**:

| Operation on the artifact's path | Legacy (bind mount) | Init-less (symlink) |
| --- | --- | --- |
| `cat`, `open()`, `read()`, `tar`, `cp`, shell redirection | regular file contents | follows symlink → same contents |
| Permissions / ownership | target file's | target file's (symlink has no separate mode) |
| Writes through the path | land in the shared emptyDir | land in the shared emptyDir (via symlink) |
| `ls -l` / `stat` (default) | regular file | **symlink** pointing at `/argo/inputs/artifacts/<name>` |
| `lstat`, `readlink` | fails / reports regular file | returns the symlink target |
| `rm` the path | removes the bind-mounted view | removes the symlink only; the artifact file remains in the shared emptyDir |
| `mv` the path elsewhere | fails / undefined | relocates the symlink (target unchanged) |

For the common cases — reading the artifact, feeding it to a program, extracting a tar, etc. — behavior is byte-for-byte identical. If your workflow code calls `lstat`/`readlink` on an input artifact path, or removes/renames the path, it will observe the difference. Overlapping user volumes are unaffected: supervisor writes those directly into the volume and no symlink is created.

An input artifact path that is an **ancestor** of a declared volume mount (e.g. artifact `path: /data` with a volume mounted at `/data/shared`) is **rejected at admission**: the controller fails pod creation with a clear error, because staging the artifact would clear the artifact's path and recurse into the mounted volume.

When the emissary stages an artifact it does not blindly clear the artifact's path:

- If **nothing exists** at the path, it creates the symlink — even when the path resolves into a mounted volume. Creating a symlink can never destroy data (it only ever creates), so an artifact deliberately pointed into a volume is written there.
- If **something already exists** at the path, it replaces that content (clearing it first, the way the legacy bind mount shadows image content) **only when doing so is safe**. It resolves any symlinks in the path's parent directories and, if the real target lands inside a declared user volume, refuses — failing the step with a clear error rather than destroying live volume data. This catches overlaps the controller cannot see at admission, such as a user image that ships a symlink (`/data → /mnt/vol`) at or above the artifact's path.

An artifact path *inside* a declared mount (`path: /data/shared/file`) is the ordinary overlap case and works: the artifact is written into the volume by `supervisor` and no symlink is created.

When an input artifact path is **also an output artifact path** (read an artifact, transform it, write the result back to the same path), the produced output is captured correctly regardless of how `main` writes it — overwriting through the symlink, or replacing the symlink via `rm` + recreate or the idiomatic write-temp-then-`rename`. The emissary inside `main` stages the live file from `main`'s own filesystem before `supervisor` collects it, so the original input is never mistaken for the output.

### `readOnlyRootFilesystem` is incompatible with init-less input artifacts

Because init-less mode creates the symlink *inside `main`'s own filesystem* — it creates the parent directory and the symlink at the artifact's path (removing any pre-existing content first when it is replacing it) — those operations require the path to be on a **writable** filesystem. If `main`'s container sets `securityContext.readOnlyRootFilesystem: true` (a common Pod Security / policy hardening) and the artifact's path falls on the read-only root filesystem, the symlink step fails with `EROFS` and the step errors before the user command runs. The same workflow succeeds in legacy mode, because there kubelet establishes the input artifact's `SubPath` bind mount at the mount-namespace level before the container starts — independent of whether the container's root filesystem is read-only.

To use input artifacts under `readOnlyRootFilesystem: true` in init-less mode, mount a **writable volume** (e.g. an `emptyDir`) at the directory that contains each artifact's path, so the symlink lands on a writable mount rather than the read-only root. Artifacts whose path already sits under a user-provided volume are unaffected. If neither is possible, run the affected template in legacy mode.

## Plugin author notes

Artifact plugins continue to use the same `driver.ArtifactDriver` interface. The only lifecycle difference in init-less mode is that the plugin container runs for the full duration of the pod (as a regular sidecar), and both Load and Save are invoked by `supervisor` on the same gRPC socket. Plugins written to run as long-lived sidecars (as the output-plugin path already does in legacy mode) work unchanged.

## Open questions / known gaps

- **Latency**: in legacy mode, plugin Load runs in parallel init containers. In init-less mode, `supervisor` parallelizes plugin loads with non-plugin loads, so per-pod latency should be equivalent to or better than legacy. If you observe a regression, please file an issue.
- **Plugin failure granularity**: the controller cannot currently distinguish "supervisor failed during plugin Load" from "supervisor failed during plugin Save" — both surface as supervisor-container failures. The `/var/run/argo/status` marker contents and supervisor logs are the authoritative source.
- **`readOnlyRootFilesystem` + input artifacts**: init-less delivers input artifacts by symlinking into `main`'s own filesystem, so `readOnlyRootFilesystem: true` makes the symlink fail with `EROFS` when the artifact's path is on the read-only root. Legacy mode is unaffected (kubelet bind mount). See [`readOnlyRootFilesystem` is incompatible with init-less input artifacts](#readonlyrootfilesystem-is-incompatible-with-init-less-input-artifacts) for the workaround.
