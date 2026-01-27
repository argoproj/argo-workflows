Component: General
Issues: 15232
Description: Add optional UID query parameter to GetWorkflow
Author: [Eduardo Rodrigues](https://github.com/eduardodbr)

Adds support for an optional uid query parameter to the existing GetWorkflow API endpoint (`/api/v1/workflows/{namespace}/{name}`). This enables more precise workflow identification, particularly useful when accessing archived workflows that may have been recreated with the same name. This implementation uses a query parameter approach (`?uid=...`), this ensures full backward compatibility with all existing endpoints.