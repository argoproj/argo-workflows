Component: General
Issues: 14661
Description: Use a string patch for PodMetadata workflow template value to allow easy parameterization.
Author: [alyssacgoins](https://github.com/alyssacgoins)

A new `PodMetadataPatch` field was added to the `WorkflowSpec` struct. This field works 
analogous to `PodSpecPatch` in that it allows for the parameterization of non-string values. 