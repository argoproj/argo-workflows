# Offloading Large Workflows

> v2.4 and after

Argo stores workflows as Kubernetes resources (i.e. within EtcD). This creates a limit to their size as resources must be under 1MB. Each resource includes the status of each node, which is stored in the `/status/nodes` field for the resource. This can be over 1MB. If this happens, we try and compress the node status and store it in `/status/compressedNodes`. If the status is still too large, we then try and store it in an SQL database.

To enable this feature, configure a Postgres, MySQL, or MariaDB database under `persistence` in [your configuration](workflow-controller-configmap.yaml) and set `nodeStatusOffLoad: true`.

## Node Status Compression

> v4.1 and after

By default node statuses are compressed with gzip. The `WORKFLOW_COMPRESSION_ALGORITHM` environment variable on the controller selects `gzip`, `zstd`, or `brotli` instead, and `WORKFLOW_COMPRESSION_LEVEL` tunes the chosen algorithm. Decompression always auto-detects the algorithm, so the variables only affect newly written statuses.

!!! Warning
    Workflows written with a non-`gzip` algorithm cannot be read by Argo versions without support for it (including `kubectl`-level tooling that gunzips `/status/compressedNodes` directly). Only change the algorithm after all components are upgraded, and be aware that downgrading past that support leaves in-flight large workflows unreadable.

Levels are algorithm-specific and default to each library's own default: `gzip` 1–9 (default 6), `zstd` 1–4 (default 2), `brotli` 0–11 (default 6).

For context, measurements on synthetic node statuses (sizes are relative to the gzip default, compression times are for ~1MiB and ~10MiB of node status JSON):

| Algorithm | Level       | Size vs gzip | Compress 1MiB | Compress 10MiB |
|-----------|-------------|--------------|---------------|----------------|
| `gzip`    | 6 (default) | 100%         | 5ms           | 43ms           |
| `zstd`    | 2 (default) | 88%          | 6ms           | 49ms           |
| `zstd`    | 3           | 82%          | 6ms           | 58ms           |
| `brotli`  | 6 (default) | 72%          | 11ms          | 129ms          |
| `brotli`  | 9           | 67%          | 24ms          | 232ms          |
| `brotli`  | 11          | 59%          | 1.7s          | 18.3s          |

Decompression speed is roughly equal for all three algorithms (~9ms per 1MiB of JSON). Higher levels than those shown buy little: brotli 11 compresses best but is ~80× slower than brotli 9, which matters because the controller re-compresses the status on every update of a large workflow. `zstd` 3 and `brotli` 9 are good choices when smaller statuses are worth slightly more controller CPU; they raise the effective node-count ceiling before offloading is required by ~20% and ~50% respectively.

These numbers are from synthetic data; real workflows may compress differently.

## FAQ

### Why aren't my workflows appearing in the database?

Offloading is expensive and often unnecessary, so we only offload when we need to. Your workflows aren't probably large enough.

### Error `Failed to submit workflow: etcdserver: request is too large.`

You must use the Argo CLI having exported `export ARGO_SERVER=...`.

### Error `offload node status is not supported`

Even after compressing node statuses, the workflow exceeded the EtcD
size limit. To resolve, either enable node status offload as described
above or look for ways to reduce the size of your workflow manifest:

- Use `withItems` or `withParams` to consolidate similar templates into a single parametrized template
- Use [template defaults](template-defaults.md) to factor shared template options to the workflow level
- Use [workflow templates](workflow-templates.md) to factor frequently-used templates into separate resources
- Use [workflows of workflows](workflow-of-workflows.md) to factor a large workflow into a workflow of smaller workflows

## Container Arguments Offloading

> v3.7 and after

When container arguments are extremely large, Argo automatically offloads them to avoid exceeding system limits. This feature addresses two types of argument size issues:

### How It Works

#### 1. ConfigMap Offloading (over 128KB total args)

If a container's JSON marshaled arguments exceed 128KB (131,072 bytes), Argo stores them in a ConfigMap instead of directly in the pod specification:

- Args are stored in a ConfigMap with key `ARGO_CONTAINER_ARGS_FILE`
- The ConfigMap is mounted as a volume at `/argo/config/`
- An environment variable `ARGO_CONTAINER_ARGS_FILE` points to `/argo/config/ARGO_CONTAINER_ARGS_FILE`
- The emissary executor automatically reads and applies the args at runtime
- Container args in the pod spec are cleared (set to nil)

This happens automatically and transparently - no workflow changes needed.

#### 2. Individual Argument Offloading (over 128KB per arg) with @filename Syntax

Even after loading args from the ConfigMap, individual arguments exceeding 128KB (131,072 bytes) would still trigger the exec syscall's "argument list too long" error (E2BIG). To handle this:

- Each argument larger than 128KB (131,072 bytes) is written to `/tmp/argo_arg_N.txt`
- The argument is replaced with `@/tmp/argo_arg_N.txt`
- **Downstream programs must support the `@filename` syntax** to read the content from the file

### Downstream Program Requirements

For individual arguments exceeding 128KB, Argo replaces the argument value with `@/tmp/argo_arg_N.txt`. **To enable your container program to handle arguments larger than 128KB**, implement file reference expansion by:

1. Detecting arguments starting with `@`
2. Reading the file path after the `@` prefix
3. Using the file contents as the actual argument value

If your program doesn't support this pattern, you'll need to either:

- Add file expansion logic to your program
- Reduce argument sizes below 128KB
- Use alternative input methods (environment variables, mounted ConfigMaps, etc.)

### Logging

When offloading occurs, you'll see these log messages:

**Controller logs:**

```text
Offloaded container args to configmap. Args >128KB will use @filename syntax container=my-container
```

**Emissary executor logs:**

```text
Reading container args from file argsFile=/argo/config/ARGO_CONTAINER_ARGS_FILE
Loaded container args from file count=5
Offloaded large argument to file. Downstream program must support @filename syntax argIndex=1 size=140000 filePath=/tmp/argo_arg_1.txt
```

### Troubleshooting

#### Error: Program doesn't recognize @filename argument

Your program doesn't support the `@filename` syntax. Options:

1. Add file expansion logic to detect `@` prefix and read the referenced file
2. Reduce the argument size below 128KB
3. Use alternative input methods (environment variables, mounted ConfigMaps, etc.)

#### Args not being offloaded when expected

- ConfigMap offloading triggers at 128KB total args (131,072 bytes, JSON marshaled)
- Individual arg file offloading triggers at 128KB per argument (131,072 bytes)
- Check controller logs for "Offloaded container args to ConfigMap" message
- Check emissary logs for "Offloaded large argument to file" message
