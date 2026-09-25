Description: Strip managedFields from controller informer caches to reduce memory usage
Authors: [Alan Clucas](https://github.com/Joibel)
Component: General
Issues: 16564

The workflow-controller now removes `metadata.managedFields` from objects before storing them in its informer caches.
`managedFields` can be 20% or more of a cached object and nothing reads it, so this reduces controller memory usage at scale with no functional change.
Objects stored in the cluster are unaffected.
