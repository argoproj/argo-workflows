Description: Configurable STS token expiration for S3 artifacts
Authors: [Jose M. Abuin](https://github.com/jmabuin)
Component: General
Issues: 15044

When using `roleARN` or IRSA (`useSDKCreds: true`) authentication for S3 artifacts, the obtained token can expire while downloading artifacts, as it is using the default expiration time.

This feature adds the `tokenExpirationInMinutes` field to the S3 artifact repository, letting users set the desired token expiration in minutes (15 to 720), for example:

    s3:
      roleARN: arn:aws:iam::123456789012:role/my-role
      tokenExpirationInMinutes: 120

Note that AWS STS will not issue a token longer than the assumed role's maximum session duration, which defaults to 60 minutes unless increased, so values above that limit are rejected.