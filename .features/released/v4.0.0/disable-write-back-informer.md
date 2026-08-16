Description: Disable write back informer by default
Author: [Eduardo Rodrigues](https://github.com/eduardodbr)
Component: General
Issues: 12352

Update the controller’s default behavior to disable the write-back informer. We’ve seen several cases of unexpected behavior that appear to be caused by the write-back mechanism, and Kubernetes docs recommend avoiding writes to the informer store. Although turning it off may increase the frequency of 409 Conflict errors, it should help reduce unpredictable controller behavior.

