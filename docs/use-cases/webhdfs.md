# Using webHDFS protocol via HTTP artifacts

webHDFS is a protocol allowing to access Hadoop or similar a data storage via a unified REST API (<https://hadoop.apache.org/docs/r3.3.3/hadoop-project-dist/hadoop-hdfs/WebHDFS.html>).

## Input Artifacts

In order to use the webHDFS protocol we will make use of HTTP artifacts, where the URL will be set to the webHDFS endpoint including the file path and all its query parameters. Suppose, our webHDFS endpoint is available under `https://mywebhdfsprovider.com/webhdfs/v1/` and we have a file `my-art.txt` located in a `data` folder, which we want to use as an input artifact. To construct the HTTP URL we need to append the file path to the base webHDFS endpoint and set the [OPEN operation](https://hadoop.apache.org/docs/r3.3.3/hadoop-project-dist/hadoop-hdfs/WebHDFS.html#Open_and_Read_a_File) in the HTTP URL parameter. This results in the following URL: `https://mywebhdfsprovider.com/webhdfs/v1/data/my-art.txt?op=OPEN`. This is all you need for webHDFS input artifacts to work! Now, when run, the workflow will download the specified webHDFS artifact into the given `path`. There are some additional fields that can be set for HTTP artifacts (e.g. HTTP headers), which you can find in the [full webHDFS example](https://github.com/argoproj/argo-workflows/blob/master/examples/webhdfs-input-output-artifacts.yaml).

```yaml
spec:
  [...]
  inputs:
    artifacts:
    - name: my-art
    path: /my-artifact
    http:
      url: "https://mywebhdfsprovider.com/webhdfs/v1/file.txt?op=OPEN"
```

## Output Artifacts

In order to declare a webHDFS output artifact, little change is necessary: We only need to change the webHDFS operation to the [CREATE operation](https://hadoop.apache.org/docs/r3.3.3/hadoop-project-dist/hadoop-hdfs/WebHDFS.html#Create_and_Write_to_a_File) and set the file path to where we want the output artifact to be stored. In this example we want to store the artifact under `outputs/newfile.txt`. We also supply the optional overwrite parameter `overwrite=true` to allow overwriting existing files in the webHDFS provider's data storage. If the `overwrite` flag is unset, the default behavior is used, which depends on the particular webHDFS provider. Below shows the example output artifact:

```yaml
spec:
  [...]
  outputs:
    artifacts:
    - name: my-art
    path: /my-artifact
    http:
      url: "https://mywebhdfsprovider.com/webhdfs/v1/outputs/newfile.txt?op=CREATE&overwrite=true"
```

## Authentication

Above example showed a minimal use case without any authentication. However, in a real-world scenario, you may want to provide some authentication option. Currently, Argo Workflows' HTTP artifacts support the following authentication mechanisms:

- HTTP Basic Auth
- OAuth2
- Client Certificates

Hence, the authentication mechanism that can be used for webHDFS artifacts are limited to those supported by HTTP artifacts. Examples for the latter two authentication mechanisms can be found in the [webHDFS example file](https://github.com/argoproj/argo-workflows/blob/master/examples/webhdfs-input-output-artifacts.yaml).

**Limitation**: Apache Hadoop itself only supports authentication via Kerberos SPNEGO and Hadoop delegation token (see <https://hadoop.apache.org/docs/r3.3.3/hadoop-project-dist/hadoop-hdfs/WebHDFS.html#Authentication>). While the former one is currently not supported for HTTP artifacts a usage of delegation tokens can be realized by supplying the authentication token in the HTTP URL of the respective input or output artifact.
