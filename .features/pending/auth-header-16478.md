Description: Add support for trusted header authentication.
Authors: [Pradeep Sagitra](https://github.com/PradeepSD476)
Component: General
Issues: 16478

Trusted header authentication allows Argo Workflows to authenticate users through HTTP headers provided by a trusted authentication proxy.
User identity, email, username, and group claims can be resolved from configured headers.
The resulting claims can be used with the existing RBAC authorization flow.
