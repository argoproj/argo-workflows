Description: Allow any claim of the ID token in SSO RBAC rules
Authors: [Matthias Bhend](https://github.com/matbhe)
Component: General
Issues: 15231 8934

`workflows.argoproj.io/rbac-rule` expressions can now use every claim of the ID token, not just the ones Argo Server models explicitly.
Previously only `sub`, `email`, `groups`, `name`, `preferred_username`, `iss`, `aud` and `exp` were available, and any other claim failed with `failed to evaluate rule: unknown name <claim>`.
This is useful for identity providers that identify users with a provider specific claim, for example a corporate user ID in `user_name`, which can now be matched with a rule such as `workflows.argoproj.io/rbac-rule: "user_name == 'my-user'"`.
Claims that Argo Server normalizes itself keep taking precedence over the raw ones.
In particular `groups` is always the normalized value, which may be derived from `customGroupClaimName` or `userInfoPath`.
Existing rules are unaffected.
