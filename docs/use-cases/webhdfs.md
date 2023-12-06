# webHDFS via HTTP artifacts

[webHDFS](https://hadoop.apache.org/docs/r3.3.3/hadoop-project-dist/hadoop-hdfs/WebHDFS.html) is a protocol allowing to access Hadoop or similar data storage via a unified REST API.

## Input Artifacts

You can use [HTTP artifacts](../walk-through/hardwired-artifacts.md) to connect to webHDFS, where the URL will be the webHDFS endpoint including the file path and any query parameters.
Suppose your webHDFS endpoint is available under `https://mywebhdfsprovider.com/webhdfs/v1/` and you have a file `my-art.txt` located in a `data` folder, which you want to use as an input artifact. To construct the URL, you append the file path to the base webHDFS endpoint and set the [OPEN operation](https://hadoop.apache.org/docs/r3.3.3/hadoop-project-dist/hadoop-hdfs/WebHDFS.html#Open_and_Read_a_File) via query parameter. The result is: `https://mywebhdfsprovider.com/webhdfs/v1/data/my-art.txt?op=OPEN`.
See the below Workflow which will download the specified webHDFS artifact into the specified `path`:

```yaml
spec:
  # ...
  inputs:
    artifacts:
    - name: my-art
    path: /my-artifact
    http:
      url: "https://mywebhdfsprovider.com/webhdfs/v1/file.txt?op=OPEN"
```

Additional fields can be set for HTTP artifacts (for example, headers). See usage in the [full webHDFS example](https://github.com/argoproj/argo-workflows/blob/main/examples/webhdfs-input-output-artifacts.yaml).

## Output Artifacts

To declare a webHDFS output artifact, instead use the [CREATE operation](https://hadoop.apache.org/docs/r3.3.3/hadoop-project-dist/hadoop-hdfs/WebHDFS.html#Create_and_Write_to_a_File) and set the file path to your desired location.
In the below example, the artifact will be stored at `outputs/newfile.txt`. You can [overwrite](https://hadoop.apache.org/docs/r3.3.3/hadoop-project-dist/hadoop-hdfs/WebHDFS.html#Overwrite) existing files with `overwrite=true`.

```yaml
spec:
  # ...
  outputs:
    artifacts:
    - name: my-art
    path: /my-artifact
    http:
      url: "https://mywebhdfsprovider.com/webhdfs/v1/outputs/newfile.txt?op=CREATE&overwrite=true"
```

## Authentication

The above examples show minimal use cases without authentication. However, in a real-world scenario, you may want to use authentication.
The authentication mechanism is limited to those supported by HTTP artifacts:

- HTTP Basic Auth
- OAuth2
- Client Certificates

Examples for the latter two mechanisms can be found in the [full webHDFS example](https://github.com/argoproj/argo-workflows/blob/main/examples/webhdfs-input-output-artifacts.yaml).

!!! Warning "Provider dependent"
    While your webHDFS provider may support the above mechanisms, Hadoop _itself_ only supports [authentication](https://hadoop.apache.org/docs/r3.3.3/hadoop-project-dist/hadoop-hdfs/WebHDFS.html#Authentication) via Kerberos SPNEGO and Hadoop delegation token. HTTP artifacts do not currently support SPNEGO, but delegation tokens can be used via the `delegation` query parameter.
