Description: Recover resource template executor pods after infrastructure failures
Authors: [Andre Kurait](https://github.com/AndreKurait)
Component: General
Issues: 15641

Resource templates using `create` or `get` recover from a missing or infrastructure-failed executor pod and continue monitoring their resource.
Recovery is independent of `failedPodRestart`, allows three recorded recovery attempts per node for failed or missing running executors, and uses deterministic resource names for `create` manifests with `generateName`.
