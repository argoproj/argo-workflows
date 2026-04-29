Description: Support for AWS RDS PostgreSQL IAM authentication
Authors: [isubasinghe](https://github.com/isubasinghe)
Component: General
Issues: 15834

Add support for authenticating to AWS RDS PostgreSQL using IAM authentication tokens.
This allows Argo Workflows to use IAM roles to connect to its persistence database, removing the need for long-lived database passwords.

  - Uses the default AWS credential chain for seamless authentication in AWS environments.
  - Supports optional region override (auto-detected if omitted).
  - Integrates with both persistence and synchronization database configurations.
