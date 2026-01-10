Component: General
Issues: 15232
Description: Add new endpoint `/workflow/{namespace}/{name}/{uid}` to filter workflows by UID.
Author: [Eduardo Rodrigues](https://github.com/eduardodbr)

Introduces a new API endpoint `GetWorkflowByUID` `/workflow/{namespace}/{name}/{uid}` that allows retrieving workflows using their unique identifier (UID), in addition to namespace and name. This complements the existing `GetWorkflow` endpoint and enables more precise workflow identification, particularly useful when workflows might have been archived or when exact workflow identification is required.