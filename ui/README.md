# Argo UI

![Argo Image](https://github.com/argoproj/argo/blob/master/argo.png?raw=true)

A web-based UI for the Argo Workflow engine. 

The UI has the following features:
* View live Argo Workflows running in the cluster
* View completed Argo Workflows
* View container logs


## Build, Run, & Release

1. Install Toolset: [NodeJS](https://nodejs.org/en/download/) and [Yarn](https://yarnpkg.com)
2. Install Dependencies: From your command line, navigate to the argo-ui directory and run `yarn install` to install dependencies.
3. Run: `yarn start` - starts API server and webpack dev UI server. API server uses current `kubectl` context to access workflow CRDs.
4. Build: `yarn build` - builds static resources into `./dist` directory.
5. Release: `IMAGE_NAMESPACE=argoproj IMAGE_TAG=latest DOCKER_PUSH=true yarn docker` - builds docker image and optionally push to docker registry.
