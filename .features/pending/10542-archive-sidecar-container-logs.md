Description: Archive sidecar container logs when archiving workflow logs
Author: [Shiwei Tang](https://github.com/siwet)
Component: General
Issues: 14802

When `archiveLogs` is enabled, sidecar container logs are now included in the archived logs alongside main container logs.
This ensures complete log visibility for workflows with sidecar containers after pod garbage collection.
