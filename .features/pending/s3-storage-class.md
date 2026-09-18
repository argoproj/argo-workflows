Description: Choose the S3 storage class for uploaded artifacts
Author: [workeainc](https://github.com/workeainc)
Component: General
Issues: 13053

S3 artifact outputs can now set an optional `storageClass`, such as `STANDARD_IA` or `INTELLIGENT_TIERING`. The configured value is passed through for both file and directory uploads.
