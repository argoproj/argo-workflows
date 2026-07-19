Description: Archive sidecar container logs when archiveLogs is enabled
Author: [shuangkun](https://github.com/shuangkun)
Component: General
Issues: 16483

When `archiveLogs` is enabled, sidecar container logs are now archived alongside the main container(s).
Each sidecar produces its own `<container>-logs` artifact, so sidecar output can be retrieved after the pod is gone.
