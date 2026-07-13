Description: add a streaming save path to artifact drivers, including gRPC streaming for plugins
Authors: [panicboat](https://github.com/panicboat)
Component: General
Issues: 12656

Artifact drivers can now save an output artifact from an `io.Reader` via a new `SaveStream` method, in addition to saving from a local file path.

Azure and HTTP/Artifactory stream the reader directly to the destination.
S3, GCS, OSS, and HDFS buffer the reader to a temp file and reuse their existing save path, so bucket creation, key handling, and retries are unchanged.

Artifact plugins gain an optional client-streaming `SaveStream` gRPC method plus a `GetCapabilities` method.
A plugin that advertises support receives the artifact content chunk by chunk with no intermediate temp file.
A plugin that does not implement these methods keeps working unchanged: Argo buffers the content to a temp file and calls the existing `Save`, so this is backward compatible.
