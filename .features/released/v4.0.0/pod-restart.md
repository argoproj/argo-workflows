Description: Restart pods that fail before starting
Authors: [Alan Clucas](https://github.com/Joibel)
Component: General
Issues: 12572

Automatically restart pods that fail before starting for reasons like node eviction.
This is safe to do even for non-idempotent workloads.
You need to configure this in your workflow controller configmap for it to take effect.
