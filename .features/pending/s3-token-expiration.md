Description: Added S3 token expiration time to use when available
Authors: [Jose M. Abuin](https://github.com/jmabuin)
Component: General
Issues: 15044

When using IAM or webidentity authentication for S3 artifacts, the obtained token can expire while downloading artifacts, as it is using the default expiration time. With this feature users should be able to set the desired expiration time in minutes, always inside the available range of time.
