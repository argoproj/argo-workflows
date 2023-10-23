#syntax=docker/dockerfile:1.2
ARG GIT_COMMIT=unknown
ARG GIT_TAG=unknown
ARG GIT_TREE_STATE=unknown
ARG BUILDER_IMAGE=golang:1.21-alpine3.18
ARG UI_IMAGE=node:20-alpine
ARG DISTROLESS_IMAGE=gcr.io/distroless/static

FROM $BUILDER_IMAGE as builder

RUN apk update && \
    (command -v go >/dev/null 2>&1 || apk add --no-cache go) && \
    apk add --no-cache \
    git \
    make \
    ca-certificates \
    wget \
    curl \
    gcc \
    bash \
    mailcap

WORKDIR /go/src/github.com/argoproj/argo-workflows
COPY go.mod .
COPY go.sum .
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

####################################################################################################

FROM $UI_IMAGE as argo-ui

RUN apk update && \
    (command -v node >/dev/null 2>&1 || apk add --no-cache nodejs-current) && \
    (command -v yarn >/dev/null 2>&1 || apk add --no-cache yarn) && \
    apk add --no-cache \
    git

COPY ui/package.json ui/yarn.lock ui/

RUN --mount=type=cache,target=/root/.yarn \
  YARN_CACHE_FOLDER=/root/.yarn JOBS=max \
  yarn --cwd ui install --network-timeout 1000000

COPY ui ui
COPY api api

RUN --mount=type=cache,target=/root/.yarn \
  YARN_CACHE_FOLDER=/root/.yarn JOBS=max \
  NODE_OPTIONS="--openssl-legacy-provider --max-old-space-size=2048" JOBS=max yarn --cwd ui build

####################################################################################################

FROM builder as argoexec-build

ARG GIT_COMMIT
ARG GIT_TAG
ARG GIT_TREE_STATE

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build make dist/argoexec GIT_COMMIT=${GIT_COMMIT} GIT_TAG=${GIT_TAG} GIT_TREE_STATE=${GIT_TREE_STATE}

####################################################################################################

FROM builder as workflow-controller-build

ARG GIT_COMMIT
ARG GIT_TAG
ARG GIT_TREE_STATE

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build make dist/workflow-controller GIT_COMMIT=${GIT_COMMIT} GIT_TAG=${GIT_TAG} GIT_TREE_STATE=${GIT_TREE_STATE}

####################################################################################################

FROM builder as argocli-build

ARG GIT_COMMIT
ARG GIT_TAG
ARG GIT_TREE_STATE

RUN mkdir -p ui/dist
COPY --from=argo-ui ui/dist/app ui/dist/app

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build STATIC_FILES=true make dist/argo GIT_COMMIT=${GIT_COMMIT} GIT_TAG=${GIT_TAG} GIT_TREE_STATE=${GIT_TREE_STATE}

####################################################################################################

FROM $DISTROLESS_IMAGE as argoexec

COPY --from=argoexec-build /go/src/github.com/argoproj/argo-workflows/dist/argoexec /bin/
COPY --from=argoexec-build /etc/mime.types /etc/mime.types
COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/

ENTRYPOINT [ "argoexec" ]

####################################################################################################

FROM $DISTROLESS_IMAGE as workflow-controller

USER 8737

COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/
COPY --chown=8737 --from=workflow-controller-build /go/src/github.com/argoproj/argo-workflows/dist/workflow-controller /bin/

ENTRYPOINT [ "workflow-controller" ]

####################################################################################################

FROM $DISTROLESS_IMAGE as argocli

USER 8737

WORKDIR /home/argo

COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/
COPY --from=argocli-build /go/src/github.com/argoproj/argo-workflows/dist/argo /bin/

ENTRYPOINT [ "argo" ]
