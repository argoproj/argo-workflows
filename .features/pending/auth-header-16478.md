Description: Add support for trusted header authentication.
Authors: [Pradeep Sagitra](https://github.com/PradeepSD476)
Component: General
Issues: 16478 15110

Trusted header authentication lets you authenticate users through HTTP headers provided by a trusted authentication proxy.
You can configure headers to set the user's identity, email, username, and groups.
You can use the resulting claims with the existing RBAC authorization flow.

Users with an empty groups list now keep the `groups` claim, so RBAC rules that reference `groups` evaluate for them.
This applies to header users and to SSO users whose identity provider sends an empty groups list.
Review RBAC rules that use negation, such as `!("contractors" in groups)`, because they can now match these users.