Description: Configurable node status compression algorithm (gzip, zstd, or brotli) and level
Authors: [Isitha Subasinghe](https://github.com/isubasinghe)
Component: General
Issues: 16262

Large workflow node statuses can now be compressed with `zstd` or `brotli` instead of gzip via the `WORKFLOW_COMPRESSION_ALGORITHM` environment variable, with `WORKFLOW_COMPRESSION_LEVEL` tuning the level.
Decompression auto-detects the algorithm.
See [node status compression](offloading-large-workflows.md#node-status-compression).
