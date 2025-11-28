Description: Support metadata.name= and metadata.name!= in field selectors
Authors: [Miltiadis Alexis](https://github.com/miltalex)
Component: General
Issues: #13468

<!--
Optional
Additional details about the feature written in markdown, aimed at users who want to learn about it
* Explain when you would want to use the feature
* Include code examples if applicable
  * Provide working examples
  * Format code using back-ticks
* Use Kubernetes style
* One sentence per line of markdown
-->

Currently, our system only supports the '=' operator for the field selector metadata.name when interacting with Kubernetes resources. This limitation restricts the flexibility and functionality of our queries. The goal of this PR is to expand support to include the '==' and '!=' operators, aligning our capabilities with native Kubernetes functionality. This enhancement will provide users with more granular control over resource selection and filtering, improving overall system usability and compatibility with standard Kubernetes practices.
