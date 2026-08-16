Description: Support metadata.name= and metadata.name!= in field selectors
Authors: [Miltiadis Alexis](https://github.com/miltalex)
Component: General
Issues: #13468

Field selectors for `metadata.name` now support the `==` and `!=` operators, giving you more flexible control over resource filtering.

Use the `==` operator to match resources with an exact name, or use `!=` to exclude resources by name.

This brings field selector behavior in line with native Kubernetes functionality and enables more precise resource queries.
