Description: Retry the initial database connection with backoff instead of crash-looping while the database starts up
Author: [Jojin](https://github.com/jojinkb)
Component: General
Issues: 8797

The workflow controller and argo-server previously exited immediately when the persistence database was not yet reachable at startup, causing them to "flap" (restart repeatedly) while waiting for PostgreSQL or MySQL to come up.
The initial database connection is now retried with backoff using the same policy as reconnections, configurable via `persistence.reconnectionConfig`.
Non-transient errors, such as invalid configuration or credentials, still fail fast.
