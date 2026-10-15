Description: Allows offloading spec.templates to the db allowing for arbitrarily large workflows
Authors: [Martin Hnatek](https://github.com/hnatekmar)
Component: General
Issues: 7121

Workflows whose inline templates exceed a configurable size threshold have those templates stored in the persistence database, leaving a small stub in the Workflow object.

The feature is disabled by default; enable it with `templateOffLoad: true` under `persistence` in the controller ConfigMap.
A Postgres, MySQL, or MariaDB database must be configured.
`templateOffloadMinSize` controls the threshold in bytes (default 256 KiB).
