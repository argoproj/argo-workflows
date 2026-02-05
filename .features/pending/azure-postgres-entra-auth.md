Description: Support for Azure PostgreSQL/Entra ID authentication
Authors: [isubasinghe](https://github.com/isubasinghe)
Component: General
Issues: 123456

Add support for authenticating to Azure Database for PostgreSQL using Azure AD (Entra ID) tokens.
This allows Argo Workflows to use Azure Workload Identity or Managed Identity to connect to its persistence database, removing the need for long-lived database passwords.
* Uses `DefaultAzureCredential` for seamless authentication in Azure environments.
* Supports configurable token scopes.
* Integrates with both persistence and synchronization database configurations.
