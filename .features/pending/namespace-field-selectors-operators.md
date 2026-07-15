Description: Add `!=` and `==` operators for namespace field selector
Authors: [Miltiadis Alexis](https://github.com/miltalex)
Component: General
Issues: 13468

You can now use the `!=` and `==` operators when filtering workflows by namespace field.
This provides more flexible query capabilities, allowing you to easily exclude specific namespaces or match exact namespace values in your workflow queries.
For example, you can filter with `namespace!=kube-system` to exclude system namespaces or `namespace==production` to target only production environments.
