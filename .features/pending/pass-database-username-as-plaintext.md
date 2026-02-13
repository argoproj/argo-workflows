Component: General
Issues: 10521
Description: Allow user name for persistenence configuration to be passed in plain text
Author: [Shivaram Vasudevan](https://github.com/Shivaram-Vasudevan)

Allow the user name for the persistence configuration to be passed in plain text (via a new configuration key `userName`). Currently in the persistence configuration, the user name can only be passed as Kubernetes secrets (via `userNameSecret`). An error is thrown if both `userNameSecret` and `userName` are set or if none of them are set.

Sample usage:
```yaml
persistence: |
    # Skipping some configuration for brevity
    postgresql:
      host: localhost
      port: 5432
      database: postgres
      tableName: argo_workflows
      userName: your-database-username
      passwordSecret:
        name: argo-postgres-config
        key: password
      ssl: true
      sslMode: require
```
