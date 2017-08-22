## AX Platform

This is production repository for AX platform code. Refer to for more information.

### Build

To build everything in `Platform` directory, run the build script. It requires a cluster configuration spec file as argument.
All cluster spec files are currently glorified bash script with environment variables. They are in `config/cluster` directory.
This script will build all required container images and push them to dev docker container registry.

```bash
$ scripts/build.sh config/cluster/test-cluster
```
