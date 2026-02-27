Description: improve S3 upload speed with customization of S3 upload threads / partsize
Authors: [Antoine Tran](https://github.com/antoinetran)
Component: General
Issues: 15636

By default, the S3 upload code is configured with 4 threads and no fix part size. The part size is dynamically set by MinIO depending on the max number of chunks and file size, but generally it is 16MiB (for file size <= 156GiB).

The feature will allow setting the number of threads and part size with env var ARGO_AGENT_S3_UPLOAD_THREADS and ARGO_AGENT_S3_UPLOAD_PART_SIZE_MIB.
