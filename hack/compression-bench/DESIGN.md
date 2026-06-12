# Node-Status Compression Benchmark

## Goal

Measure whether replacing the current node-status compression (JSON → gzip →
base64, `workflow/packer`) with proto serialization and/or zstd with a trained
dictionary is worth a format migration. This is a measurement prototype only:
no changes to `workflow/packer` or anything shipped. The numbers decide
whether an integration phase happens later.

## Hypothesis to settle

zstd dictionaries help most on small payloads, but packer compression only
triggers above `MAX_WORKFLOW_SIZE` (1MB default), where a blob already carries
plenty of self-redundancy inside the LZ window. The dictionary may therefore
help least exactly where it is used most. Proto-instead-of-JSON (keys become
field tags) should help at every scale. The per-scale results table settles
both claims.

## Shape

A standalone Go CLI at `hack/compression-bench/`, run via
`go run ./hack/compression-bench`. Three files:

- `main.go` — flag parsing, orchestration, results table to stdout.
- `corpus.go` — corpus synthesis from `examples/`.
- `codecs.go` — the codec matrix and round-trip verification.

## Corpus synthesis

- Parse `examples/*.yaml`, keep only `kind: Workflow` documents.
- For each spec, walk its templates and synthesize a plausible
  `map[string]NodeStatus`: realistic node IDs and names, phases (mostly
  `Succeeded`, some `Failed`/`Running`), start/finish timestamps, boundary
  IDs, children links, template names, and inputs/outputs with parameters.
- A fan-out multiplier scales each spec to target node counts of roughly
  100, 1k, 5k, and 10k nodes.
- All randomness is seeded deterministically so runs are reproducible.
- Train/eval split: about half the synthesized blobs train the dictionary;
  only the held-out half is measured. Dictionary numbers on training data
  would be overfit and meaningless.

## Codec matrix

| Codec | Serialization | Compression |
|---|---|---|
| `json+gzip` (baseline) | JSON | gzip via `util/file.CompressEncodeString` (current packer path) |
| `json+zstd` | JSON | zstd |
| `json+zstd+dict` | JSON | zstd with dictionary trained on JSON blobs |
| `proto+zstd` | gogo proto | zstd |
| `proto+zstd+dict` | gogo proto | zstd with dictionary trained on proto blobs |

- Proto serialization wraps the nodes map as
  `wfv1.WorkflowStatus{Nodes: ...}` and uses the existing generated gogo
  `Marshal`/`Unmarshal`.
- zstd encoder/decoder and dictionary training come from
  `github.com/klauspost/compress` (already in the module graph at v1.18.6;
  promoted from indirect to direct). Training uses the pure-Go
  `dict.BuildZstdDict` — no zstd CLI, no cgo.
- Every codec round-trips: decode output must be semantically equal to the
  input nodes map (`assert`-style deep equality in the harness). A fidelity
  bug must fail the run, not masquerade as a good ratio.

## Metrics

Per corpus scale × codec:

- compressed size (raw bytes)
- base64-encoded size (what actually lands in the CRD string field / etcd)
- ratio vs the `json+gzip` baseline
- encode and decode wall time
- dictionary size (for dict codecs; the dictionary ships with readers, so it
  counts as a cost)

Output is a plain text table to stdout.

## Non-goals

- No changes to `workflow/packer`, `util/file`, or any shipped code path.
- No format versioning / magic-byte design — that belongs to a later
  integration phase, only if these numbers justify it.
- No cluster runs; the corpus is synthesized.

## Success criteria

The harness runs from a clean checkout with no external tools and prints a
table that answers: (1) how much smaller than gzip each codec gets at each
scale, and (2) whether the trained dictionary still earns its keep at the
1MB+ sizes where packer compression actually fires.
