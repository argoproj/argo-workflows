Description: Add parallelism configuration options for S3 artifact uploads and downloads to improve performance for many files and large files
Authors: [Your Name](https://github.com/yourusername)
Component: General
Issues: 12442 9022 4014

This feature adds configurable parallelism options for S3 artifact operations to significantly improve performance when dealing with:
* Many small files (parallel directory uploads/downloads)
* Large single files (multipart uploads/downloads with multiple threads)

### Configuration Options

The feature can be configured either via artifact repository configuration or inline on individual artifacts.

#### Artifact Repository Configuration

Configure in the artifact repository ConfigMap:

```yaml
s3:
  enableParallelism: true
  parallelism: 10
  fileCountThreshold: 10
  fileSizeThreshold: 64Mi
  partSize: 128Mi
```

#### Inline Artifact Configuration

Configure directly on artifacts in workflow templates:

```yaml
outputs:
  artifacts:
    - name: large-file
      path: /tmp/large-file
      s3:
        enableParallelism: true
        parallelism: 5
        fileSizeThreshold: 10Mi
        partSize: 32Mi
```

### Configuration Fields

* `enableParallelism`: Enable/disable parallel operations (default: true)
* `parallelism`: Number of concurrent workers for parallel operations (default: 10)
* `fileCountThreshold`: Minimum number of files in a directory to trigger parallel operations (default: 10)
* `fileSizeThreshold`: Minimum file size to trigger multipart upload/download (default: 64Mi, supports Kubernetes resource quantity strings)
* `partSize`: Part size for multipart uploads (default: minio default, typically 128MB, supports Kubernetes resource quantity strings)

### Use Cases

* Uploading/downloading directories with many files (e.g., build artifacts, test results)
* Handling large single files (e.g., database dumps, large datasets)
* Improving workflow execution time when artifact I/O is a bottleneck
