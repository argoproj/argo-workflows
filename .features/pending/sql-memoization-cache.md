Description: Add SQL database-backed memoization cache as an alternative to ConfigMaps.
Authors: [droctothorpe](https://github.com/droctothorpe)
Component: General
Issues: 15952
PRs: 15938

The controller can now store memoization entries in a PostgreSQL or MySQL database instead of Kubernetes ConfigMaps.
The SQL backend removes the 1 MB ConfigMap size limit.
ConfigMaps remain the default; opt in by adding a `memoization` block to the `workflow-controller-configmap`.
The SQL backend stores entries in the configured table, which defaults to `cache_entries`.
The SQL backend records an expiry only when the template specifies `memoize.maxAge`; entries without it have no default expiry.
The leader garbage-collects expired entries at startup and periodically thereafter.
With either backend, the controller now marks HTTP and plugin nodes `Error` if it cannot save their memoized outputs, instead of only logging the error.
