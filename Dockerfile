#syntax=docker/dockerfile:1.2

ARG DOCKER_CHANNEL=stable
ARG DOCKER_VERSION=20.10.14
# NOTE: kubectl version should be one minor version less than https://storage.googleapis.com/kubernetes-release/release/stable.txt
ARG KUBECTL_VERSION=1.22.3
ARG JQ_VERSION=1.6

FROM golang:1.17 as builder

RUN apt-get update && apt-get --no-install-recommends install -y \
    git \
    make \
    apt-utils \
    apt-transport-https \
    ca-certificates \
    wget \
    gcc \
    libcap2-bin \
    zip && \
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

FROM alpine:3 as argoexec-base

ARG DOCKER_CHANNEL
ARG DOCKER_VERSION
ARG KUBECTL_VERSION

RUN apk --no-cache add curl procps git tar libcap jq

COPY hack/arch.sh hack/os.sh /bin/

RUN if [ $(arch.sh) = ppc64le ] || [ $(arch.sh) = s390x ]; then \
        curl -o docker.tgz https://download.docker.com/$(os.sh)/static/${DOCKER_CHANNEL}/$(uname -m)/docker-18.06.3-ce.tgz; \
    else \
        curl -o docker.tgz https://download.docker.com/$(os.sh)/static/${DOCKER_CHANNEL}/$(uname -m)/docker-${DOCKER_VERSION}.tgz; \
    fi && \
    tar --extract --file docker.tgz --strip-components 1 --directory /usr/local/bin/ && \
    rm docker.tgz
RUN curl -o /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/$(os.sh)/$(arch.sh)/kubectl && \
    chmod +x /usr/local/bin/kubectl
RUN rm /bin/arch.sh /bin/os.sh

COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/


####################################################################################################

FROM node:16 as argo-ui

COPY ui/package.json ui/yarn.lock ui/

RUN JOBS=max yarn --cwd ui install --network-timeout 1000000

COPY ui ui
COPY api api

RUN NODE_OPTIONS="--max-old-space-size=2048" JOBS=max yarn --cwd ui build

####################################################################################################

FROM builder as argoexec-build

# Tell git to forget about all of the files that were not included because of .dockerignore in order to ensure that
# the git state is "clean" even though said .dockerignore files are not present
RUN cat .dockerignore >> .gitignore
RUN git status --porcelain | cut -c4- | xargs git update-index --skip-worktree

RUN --mount=type=cache,target=/root/.cache/go-build make dist/argoexec
RUN setcap CAP_SYS_PTRACE,CAP_SYS_CHROOT+ei dist/argoexec

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

FROM argoexec-base as argoexec

COPY --from=argoexec-build /go/src/github.com/argoproj/argo-workflows/dist/argoexec /usr/local/bin/
COPY --from=argoexec-build /etc/mime.types /etc/mime.types

ENTRYPOINT [ "argoexec" ]

####################################################################################################

FROM scratch as workflow-controller

USER 8737

COPY --chown=8737 --from=workflow-controller-build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --chown=8737 --from=workflow-controller-build /go/src/github.com/argoproj/argo-workflows/dist/workflow-controller /bin/

ENTRYPOINT [ "workflow-controller" ]

####################################################################################################

FROM scratch as argocli

USER 8737

WORKDIR /home/argo

COPY hack/ssh_known_hosts /etc/ssh/
COPY hack/nsswitch.conf /etc/
COPY --from=argocli-build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=argocli-build /go/src/github.com/argoproj/argo-workflows/dist/argo /bin/

ENTRYPOINT [ "argo" ]
