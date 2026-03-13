Description: S3 virtual-hosted-style bucket addressing
Author: [Himesh Panchal](https://github.com/himeshp)
Component: Artifacts
Issues: 10851

S3 artifact storage now supports configuring the bucket addressing style via the `addressingStyle` field.
Valid values are `""` (auto-detect, default), `"path"` (force path-style), and `"virtual-hosted"` (force virtual-hosted-style).
This fixes broken log streaming and artifact browsing for S3-compatible providers that only support virtual-hosted-style addressing.
