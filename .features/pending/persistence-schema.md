Description: You can now configure a specific postgres database schema for Argo Persistence in the controller config map.
Authors: [Jonathan Pollert](https://github.com/jnt0r)
Component: General
Issues: 2452

Added support for custom database schemas to improve data isolation and security in shared environments. This allows Argo to operate within a designated logical schema rather than the default. Note: This feature is not applicable to MySQL, as it does not support logical schemas; for MySQL users, database names should continue to be used for application separation.