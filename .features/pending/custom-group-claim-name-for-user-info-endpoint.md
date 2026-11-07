Description: Add customGroupClaimName option for using user-info endpoint
Authors: [okzw999](https://github.com/okzw999)
Component: General
Issues: 15803

You can also configure both `customGroupClaimName` and `userInfoPath` to specify the user info endpoint that contains the custom claim name for groups. This allows Argo Workflows to adapt to your OIDC provider providing user information with a custom claim group name.