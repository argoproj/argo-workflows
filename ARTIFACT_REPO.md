# Configuring Your Artifact Repository

To run Argo workflows that use artifacts, you must configure and use an artifact repository.
Argo supports any S3 compatible artifact repository such as AWS, GCS and Minio.
This section shows how to configure the artifact repository. Subsequent sections will show how to use it.

## Configuring Minio

```
$ brew install kubernetes-helm # mac
$ helm init
$ helm install stable/minio --name argo-artifacts --set service.type=LoadBalancer
```

Login to the Minio UI using a web browser (port 9000) after obtaining the external IP using `kubectl`.
```
$ kubectl get service argo-artifacts
```

On Minikube:
```
$ minikube service --url argo-artifacts
```

NOTE: When minio is installed via Helm, it uses the following hard-wired default credentials,
which you will use to login to the UI:
* AccessKey: AKIAIOSFODNN7EXAMPLE
* SecretKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

Create a bucket named `my-bucket` from the Minio UI.


## Configuring AWS S3

Create your bucket and access keys for the bucket. AWS access keys have the same permissions as the user they are associated with. In particular, you cannot create access keys with reduced scope. If you want to limit the permissions for an access key, you will need to create a user with just the permissions you want to associate with the access key. Otherwise, you can just create an access key using your existing user account.

```
$ export mybucket=bucket249
$ cat > policy.json <<EOF
{
   "Version":"2012-10-17",
   "Statement":[
      {
         "Effect":"Allow",
         "Action":[
            "s3:PutObject",
            "s3:GetObject"
         ],
         "Resource":"arn:aws:s3:::$mybucket/*"
      }
   ]
}
EOF
$ aws s3 mb s3://$mybucket [--region xxx]
$ aws iam create-user --user-name $mybucket-user
$ aws iam put-user-policy --user-name $mybucket-user --policy-name $mybucket-policy --policy-document file://policy.json
$ aws iam create-access-key --user-name $mybucket-user > access-key.json
```


NOTE: if you want argo to figure out which region your buckets belong in, you must additionally set the following statement policy. Otherwise, you must specify a bucket region in your workflow configuration.

```
    ...
      {
         "Effect":"Allow",
         "Action":[
            "s3:GetBucketLocation"
         ],
         "Resource":"arn:aws:s3:::*"
      }
    ...
```

## Configuring GCS (Google Cloud Storage)
Create a bucket from the GCP Console (https://console.cloud.google.com/storage/browser).

Enable S3 compatible access and create an access key.
Note that S3 compatible access is on a per project rather than per bucket basis.
- Navigate to Storage > Settings (https://console.cloud.google.com/storage/settings).
- Enable interoperability access if needed.
- Create a new key if needed.

# Configure the Default Artifact Repository

In order for Argo to use your artifact repository, you must configure it as the default repository.
Edit the workflow-controller config map with the correct endpoint and access/secret keys for your repository.

Use the `endpoint` corresponding to your S3 provider:
- AWS: s3.amazonaws.com
- GCS: storage.googleapis.com
- Minio: my-minio-endpoint.default:9000

The `key` is name of the object in the `bucket` The `accessKeySecret` and `secretKeySecret` are secret selectors that reference the specified kubernetes secret.  The secret is expected to have have the keys 'accessKey' and 'secretKey', containing the base64 encoded credentials to the bucket.

For AWS, the `accessKeySecret` and `secretKeySecret` correspond to AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY respectively.

EC2 provides a metadata API via which applications using the AWS SDK may assume IAM roles associated with the instance. If you are running argo on EC2 and the instance role allows access to your S3 bucket, you can configure the workflow step pods to assume the role. To do so, simply omit the `accessKeySecret` and `secretKeySecret` fields.

For GCS, the `accessKeySecret` and `secretKeySecret` for S3 compatible access can be obtained from the GCP Console. Note that S3 compatible access is on a per project rather than per bucket basis.
- Navigate to Storage > Settings (https://console.cloud.google.com/storage/settings).
- Enable interoperability access if needed.
- Create a new key if needed.

For Minio, the `accessKeySecret` and `secretKeySecret` naturally correspond the AccessKey and SecretKey.

Example:
```
$ kubectl edit configmap workflow-controller-configmap -n argo		# assumes argo was installed in the argo namespace
...
data:
  config: |
    artifactRepository:
      s3:
        bucket: my-bucket
        keyPrefix: prefix/in/bucket     #optional
        endpoint: my-minio-endpoint.default:9000        #AWS => s3.amazonaws.com; GCS => storage.googleapis.com
        insecure: true                  #omit for S3/GCS. Needed when minio runs without TLS
        accessKeySecret:                #omit if accessing via AWS IAM
          name: my-minio-cred
          key: accesskey
        secretKeySecret:                #omit if accessing via AWS IAM
          name: my-minio-cred
          key: secretkey
```
The secrets are retrieve from the namespace you use to run your workflows. Note that you can specify a `keyPrefix`.

# Accessing Non-Default Artifact Repositories

This section shows how to access artifacts from non-default artifact repositories.

The `endpoint`, `accessKeySecret` and `secretKeySecret` are the same as for configuring the default artifact repository described previously.

```
  templates:
  - name: artifact-example
    inputs:
      artifacts:
      - name: my-input-artifact
        path: /my-input-artifact
        s3:
          endpoint: s3.amazonaws.com
          bucket: my-aws-bucket-name
          key: path/in/bucket/my-input-artifact.tgz
          accessKeySecret:
            name: my-aws-s3-credentials
            key: accessKey
          secretKeySecret:
            name: my-aws-s3-credentials
            key: secretKey
    outputs:
      artifacts:
      - name: my-output-artifact
        path: /my-ouput-artifact
        s3:
          endpoint: storage.googleapis.com
          bucket: my-gcs-bucket-name
          # NOTE that all output artifacts are automatically tarred and
          # gzipped before saving. So as a best practice, .tgz or .tar.gz
          # should be incorporated into the key name so the resulting file
          # has an accurate file extension.
          key: path/in/bucket/my-output-artifact.tgz
          accessKeySecret:
            name: my-gcs-s3-credentials
            key: accessKey
          secretKeySecret:
            name: my-gcs-s3-credentials
            key: secretKey
    container:
      image: debian:latest
      command: [sh, -c]
      args: ["cp -r /my-input-artifact /my-output-artifact"]
```
