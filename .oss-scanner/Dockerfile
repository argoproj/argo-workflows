# Builds Argo Workflows for the Anthropic OSS Scanner from the checkout at /src, and keeps everything needed to
# build and test it offline: the Go module cache, the UI's node_modules and built UI, and the argo (CLI and
# argo-server), workflow-controller and argoexec binaries.
# The final stage must keep a package manager, so this does not use the release images' distroless base.
FROM docker.io/library/golang:1.27-trixie

# Unit tests expect no cluster: KUBECONFIG=/dev/null is what `make test` uses.
ENV DEBIAN_FRONTEND=noninteractive \
    CGO_ENABLED=0 \
    KUBECONFIG=/dev/null

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl git make jq procps \
    && rm -rf /var/lib/apt/lists/*

# Node.js for the UI, from the official image (CI builds the UI with Node 24), and yarn classic, which the UI uses.
COPY --from=docker.io/library/node:24-trixie /usr/local/bin/node /usr/local/bin/node
COPY --from=docker.io/library/node:24-trixie /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s ../lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -s ../lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx \
    && npm install -g yarn@1.22.22 \
    && node --version && yarn --version

COPY . /src
WORKDIR /src

# Go dependencies. If go.mod asks for a newer Go than this image has, the toolchain is downloaded here and stays
# in the module cache for offline use.
RUN go mod download

# The UI is embedded into the argo binary (ui/embed.go), so build it before the Go binaries.
# yarn.lock pulls argo-ui from git, which is why git is installed above.
RUN yarn --cwd ui install --frozen-lockfile --network-timeout 600000 \
    && NODE_OPTIONS=--max-old-space-size=4096 yarn --cwd ui build \
    && test -f ui/dist/app/index.html

RUN go build -o /usr/local/bin/argo ./cmd/argo \
    && go build -o /usr/local/bin/workflow-controller ./cmd/workflow-controller \
    && go build -o /usr/local/bin/argoexec ./cmd/argoexec \
    && argo version --short

# Compile every non-test package once so later builds and reproducers are fast.
RUN go build ./...

# Baseline tests for the security-sensitive areas: argo-server auth and the servers that read with argo-server's
# own credentials, artifact handling in the server and executor, expression and template evaluation, and
# workflow validation. A failing test must not abort the build.
RUN go test -short -p 4 -timeout 20m \
      ./server/auth/... ./server/apiserver/... ./server/artifacts/... ./server/workflow/... \
      ./server/workflowtemplate/... ./server/clusterworkflowtemplate/... ./server/workflowarchive/... \
      ./server/sync/... ./server/event/... \
      ./workflow/executor/... ./workflow/artifacts/... ./workflow/validate/... \
      ./util/expr/... ./util/template/... \
    || echo 'WARNING: baseline Go tests reported failures'

# Reproducers and patches must build with the network disabled.
ENV GOPROXY=off \
    GOSUMDB=off

CMD ["bash"]
