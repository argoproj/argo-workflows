#syntax=docker/dockerfile:1.25
ARG GIT_COMMIT=unknown
ARG GIT_TAG=unknown
ARG GIT_TREE_STATE=unknown

FROM golang:1.26.1-alpine3.23 AS builder

# libc-dev to build openapi-gen
RUN apk update && apk add --no-cache \
    git \
    make \
    ca-certificates \
    wget \
    curl \
    gcc \
    libc-dev \
    bash \
    mailcap

WORKDIR /go/src/github.com/argoproj/argo-workflows
COPY go.mod .
COPY go.sum .
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

####################################################################################################

# Delve debugger, copied into the `-dev` images so the controller/server/executor
# can be run under `dlv exec` when Tilt is invoked with `--debug=...`. Pinned to a
# release that supports the builder's Go toolchain. Dev-only: never used by the
# distroless production targets.
FROM builder AS dlv-build
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/go-delve/delve/cmd/dlv@v1.27.0

####################################################################################################

FROM node:20-alpine AS argo-ui

RUN apk update && apk add --no-cache git

COPY ui/package.json ui/yarn.lock ui/

RUN --mount=type=cache,target=/root/.yarn \
  YARN_CACHE_FOLDER=/root/.yarn JOBS=max \
  yarn --cwd ui install --network-timeout 1000000

COPY ui ui
COPY api api

RUN --mount=type=cache,target=/root/.yarn \
  YARN_CACHE_FOLDER=/root/.yarn JOBS=max \
  NODE_OPTIONS="--max-old-space-size=2048" JOBS=max yarn --cwd ui build

####################################################################################################

FROM builder AS argoexec-build

ARG GIT_COMMIT
ARG GIT_TAG
ARG GIT_TREE_STATE

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build make dist/argoexec GIT_COMMIT=${GIT_COMMIT} GIT_TAG=${GIT_TAG} GIT_TREE_STATE=${GIT_TREE_STATE}

####################################################################################################

FROM builder AS workflow-controller-build

ARG GIT_COMMIT
ARG GIT_TAG
ARG GIT_TREE_STATE

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build make dist/workflow-controller GIT_COMMIT=${GIT_COMMIT} GIT_TAG=${GIT_TAG} GIT_TREE_STATE=${GIT_TREE_STATE}

####################################################################################################

FROM builder AS argocli-build

ARG GIT_COMMIT
ARG GIT_TAG
ARG GIT_TREE_STATE

RUN mkdir -p ui/dist
COPY --from=argo-ui ui/dist/app ui/dist/app
# update timestamp so that `make` doesn't try to rebuild this -- it was already built in the previous stage
RUN touch ui/dist/app/index.html

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build STATIC_FILES=true make dist/argo GIT_COMMIT=${GIT_COMMIT} GIT_TAG=${GIT_TAG} GIT_TREE_STATE=${GIT_TREE_STATE}

####################################################################################################

FROM gcr.io/distroless/static-debian13:latest@sha256:9197324ba51d9cd071af8505989365c006adf9d6d2067eada25aef00abbb5278 AS argoexec-base

COPY --from=argoexec-build /etc/mime.types /etc/mime.types
COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/

####################################################################################################

FROM argoexec-base AS argoexec-nonroot

USER 8737

COPY --chown=8737 --from=argoexec-build /go/src/github.com/argoproj/argo-workflows/dist/argoexec /bin/

ENTRYPOINT [ "argoexec" ]

####################################################################################################
FROM argoexec-base AS argoexec

COPY --from=argoexec-build /go/src/github.com/argoproj/argo-workflows/dist/argoexec /bin/

ENTRYPOINT [ "argoexec" ]

####################################################################################################

FROM gcr.io/distroless/static-debian13:latest@sha256:9197324ba51d9cd071af8505989365c006adf9d6d2067eada25aef00abbb5278 AS workflow-controller

USER 8737

COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/
COPY --chown=8737 --from=workflow-controller-build /go/src/github.com/argoproj/argo-workflows/dist/workflow-controller /bin/

ENTRYPOINT [ "workflow-controller" ]

####################################################################################################

FROM gcr.io/distroless/static-debian13:latest@sha256:9197324ba51d9cd071af8505989365c006adf9d6d2067eada25aef00abbb5278 AS argocli

USER 8737

WORKDIR /home/argo

COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/
COPY --from=argocli-build /go/src/github.com/argoproj/argo-workflows/dist/argo /bin/

ENTRYPOINT [ "argo" ]

####################################################################################################
# Dev-only stages for Tilt. Small alpine base; NOT shipped to users. The
# binaries are compiled on the host (by Tilt local_resources) and COPYed from
# the build context, so each binary is built exactly once. On change Tilt
# rebuilds these (trivial COPY) and recreates the pod.

FROM alpine:3.24 AS workflow-controller-dev
RUN apk add --no-cache ca-certificates
COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/
COPY dist/workflow-controller /bin/workflow-controller
# Delve, for `tilt up -- --debug=controller` (the Tiltfile wraps the entrypoint).
COPY --from=dlv-build /go/bin/dlv /bin/dlv
# Match the prod image's non-root user so runAsNonRoot is satisfied.
USER 8737
ENTRYPOINT [ "workflow-controller" ]

####################################################################################################

FROM alpine:3.24 AS argocli-dev
RUN apk add --no-cache ca-certificates
WORKDIR /home/argo
COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/
COPY dist/argo /bin/argo
# Delve, for `tilt up -- --debug=server` (the Tiltfile wraps the entrypoint).
COPY --from=dlv-build /go/bin/dlv /bin/dlv
USER 8737
ENTRYPOINT [ "argo" ]

####################################################################################################

FROM alpine:3.24 AS argoexec-dev
RUN apk add --no-cache ca-certificates mailcap
COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/
COPY dist/argoexec /bin/argoexec
USER 8737
ENTRYPOINT [ "argoexec" ]
