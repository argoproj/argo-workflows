#syntax=docker/dockerfile:1.2

FROM golang:1.18 as builder

RUN apt-get update && apt-get --no-install-recommends install -y \
    git \
    make \
    apt-utils \
    apt-transport-https \
    ca-certificates \
    wget \
    gcc && \
    apt-get clean \
    && rm -rf \
        /var/lib/apt/lists/* \
        /tmp/* \
        /var/tmp/* \
        /usr/share/man \
        /usr/share/doc \
        /usr/share/doc-base

WORKDIR /tmp

# https://blog.container-solutions.com/faster-builds-in-docker-with-go-1-11
WORKDIR /go/src/github.com/argoproj/argo-workflows
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

####################################################################################################

FROM node:16 as argo-ui

COPY ui/package.json ui/yarn.lock ui/

RUN JOBS=max yarn --cwd ui install --network-timeout 1000000

COPY ui ui
COPY api api

RUN NODE_OPTIONS="--max-old-space-size=2048" JOBS=max yarn --cwd ui build

####################################################################################################

FROM builder as argoexec-build

COPY hack/arch.sh hack/os.sh /bin/

# NOTE: kubectl version should be one minor version less than https://storage.googleapis.com/kubernetes-release/release/stable.txt
RUN curl -o /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v1.22.3/bin/$(os.sh)/$(arch.sh)/kubectl && \
    chmod +x /usr/local/bin/kubectl

RUN curl -o /usr/local/bin/jq https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64 && \
  chmod +x /usr/local/bin/jq

# Tell git to forget about all of the files that were not included because of .dockerignore in order to ensure that
# the git state is "clean" even though said .dockerignore files are not present
RUN cat .dockerignore >> .gitignore
RUN git status --porcelain | cut -c4- | xargs git update-index --skip-worktree

RUN --mount=type=cache,target=/root/.cache/go-build make dist/argoexec

####################################################################################################

FROM builder as workflow-controller-build

# Tell git to forget about all of the files that were not included because of .dockerignore in order to ensure that
# the git state is "clean" even though said .dockerignore files are not present
RUN cat .dockerignore >> .gitignore
RUN git status --porcelain | cut -c4- | xargs git update-index --skip-worktree

RUN --mount=type=cache,target=/root/.cache/go-build make dist/workflow-controller

####################################################################################################

FROM builder as argocli-build

RUN mkdir -p ui/dist
COPY --from=argo-ui ui/dist/app ui/dist/app
# stop make from trying to re-build this without yarn installed
RUN touch ui/dist/node_modules.marker
RUN touch ui/dist/app/index.html

# Tell git to forget about all of the files that were not included because of .dockerignore in order to ensure that
# the git state is "clean" even though said .dockerignore files are not present
RUN cat .dockerignore >> .gitignore
RUN git status --porcelain | cut -c4- | xargs git update-index --skip-worktree

RUN --mount=type=cache,target=/root/.cache/go-build make dist/argo

####################################################################################################

FROM gcr.io/distroless/static as argoexec

COPY --from=argoexec-build /usr/local/bin/kubectl /bin/
COPY --from=argoexec-build /usr/local/bin/jq /bin/
COPY --from=argoexec-build /go/src/github.com/argoproj/argo-workflows/dist/argoexec /bin/
COPY --from=argoexec-build /etc/mime.types /etc/mime.types
COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/

ENTRYPOINT [ "argoexec" ]

####################################################################################################

FROM gcr.io/distroless/static as workflow-controller

USER 8737

COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/
COPY --chown=8737 --from=workflow-controller-build /go/src/github.com/argoproj/argo-workflows/dist/workflow-controller /bin/

ENTRYPOINT [ "workflow-controller" ]

####################################################################################################

FROM gcr.io/distroless/static as argocli

USER 8737

WORKDIR /home/argo

COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/
COPY --from=argocli-build /go/src/github.com/argoproj/argo-workflows/dist/argo /bin/

ENTRYPOINT [ "argo" ]
