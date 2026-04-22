Description: Pause the running pod after container exit based on file existence or environment variable
Author: [guanguxiansheng](https://github.com/guanguxiansheng)
Component: General
Issues: 0

When the main process in the container exits, the emissary can optionally pause instead of exiting immediately.
This allows attaching to the pod for debugging (e.g. inspecting logs or the filesystem).

Pause is triggered when either the file `{varRunArgo}/ctr/{containerName}/after-pause` exists, or the environment variable `ARGO_DEBUG_PAUSE_AFTER` is set to `"true"`.

To resume and release the container, create the file `{varRunArgo}/ctr/{containerName}/after` (or remove the `after-pause` file when using file-based trigger).
