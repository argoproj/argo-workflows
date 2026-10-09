Description: Configurable STS token expiration for S3 artifacts
Authors: [Jose M. Abuin](https://github.com/jmabuin)
Component: General
Issues: 15044

When using `roleARN` or IRSA (`useSDKCreds: true`) authentication for S3 artifacts, the obtained token can expire while downloading artifacts, as it is using the default expiration time.

This feature adds the `tokenExpirationInMinutes` field to the S3 artifact repository, letting you set the desired token expiration in minutes (15 to 720), for example, setting `tokenExpirationInMinutes: 120` alongside `useSDKCreds: true` under `s3`.

Note that AWS STS doesn't issue a token longer than the assumed role's maximum session duration, which defaults to 60 minutes unless increased, so STS rejects anything above that limit.

When used with `roleARN`, the value is limited to 60 minutes if the pod already has a role (for example, IRSA), because assuming a role from another role is role chaining, which AWS caps at one hour regardless of the role's maximum session duration.
