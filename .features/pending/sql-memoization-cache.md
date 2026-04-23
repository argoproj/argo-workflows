Description: Add SQL database-backed memoization cache as an alternative to ConfigMaps.
Authors: [droctothorpe](https://github.com/droctothorpe)
Component: General
Issues: 15952
PRs: 15938

Memoization can now store cache entries in a PostgreSQL or MySQL database instead of Kubernetes ConfigMaps.
The SQL backend removes the 1 MB ConfigMap size limit and persists cache entries across cluster restarts.
ConfigMaps remain the default; opt in by adding a `memoization` block to the `workflow-controller-configmap`.
Each cache entry computes an `expires_at` timestamp at save time from the template's `maxAge` field (default: 30 days).
The default max age can be overridden via the `DEFAULT_MAX_AGE` environment variable on the controller.
A periodic garbage collector deletes expired entries whose `expires_at` has elapsed.
