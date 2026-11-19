Description: Add SQL database-backed memoization cache as an alternative to ConfigMaps.
Authors: [droctothorpe](https://github.com/droctothorpe)
Component: General
Issues: 15952
PRs: 15938

Memoization can now store cache entries in a PostgreSQL or MySQL database instead of Kubernetes ConfigMaps.
The SQL backend removes the 1 MB ConfigMap size limit.
ConfigMaps remain the default; opt in by adding a `memoization` block to the `workflow-controller-configmap`.
SQL-backed entries are stored in the configured table, which defaults to `cache_entries`.
Memoization still requires an explicit cache key.
SQL entries record an expiry only when the template specifies `memoize.maxAge`; entries without it have no default expiry.
A periodic garbage collector deletes expired entries whose `expires_at` has elapsed.
