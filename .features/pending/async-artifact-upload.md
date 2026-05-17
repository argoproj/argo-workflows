Description: Report artifact metadata before upload completes, with concurrent uploads.
Author: [daixin1204](https://github.com/SimbaKingjoe)
Component: General
Issues: 16091

This change separates artifact key generation from file uploads, allowing the
wait container to report output metadata (S3 keys, types) to the controller
immediately. Actual artifact uploads then run concurrently via goroutines with
sync.WaitGroup synchronization. This lays the foundation for full async artifact
uploading where downstream tasks that do not consume the artifacts can proceed
without waiting for upload completion.
