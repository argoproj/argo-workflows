####################################################################################################
# Builder image
# Initial stage which pulls prepares build dependencies and CLI tooling we need for our final image
# Also used as the image in CI jobs so needs all dependencies
####################################################################################################
FROM golang:1.13.4 as builder

ARG IMAGE_OS=linux
ARG IMAGE_ARCH=amd64

RUN apt-get update && apt-get --no-install-recommends install -y \
    git \
    make \
    apt-utils \
    apt-transport-https \
    ca-certificates \
    wget \
    gcc \
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

# Install docker
ENV DOCKER_CHANNEL stable
ENV DOCKER_VERSION 18.09.1

RUN if [ "${IMAGE_OS}" = "linux" -a "${IMAGE_ARCH}" = "amd64" ]; then \
    	wget -O docker.tgz https://download.docker.com/linux/static/${DOCKER_CHANNEL}/x86_64/docker-${DOCKER_VERSION}.tgz; \
    elif [ "${IMAGE_OS}" = "linux" -a "${IMAGE_ARCH}" = "arm64" ]; then \
	wget -O docker.tgz https://download.docker.com/linux/static/${DOCKER_CHANNEL}/aarch64/docker-${DOCKER_VERSION}.tgz; \
    fi && \
    tar --extract --file docker.tgz --strip-components 1 --directory /usr/local/bin/ && \
    rm docker.tgz

# Install dep
ENV DEP_VERSION=0.5.1
RUN if [ "${IMAGE_OS}" = "linux" -a "${IMAGE_ARCH}" = "amd64" ]; then \
	wget https://github.com/golang/dep/releases/download/v${DEP_VERSION}/dep-linux-amd64 -O /usr/local/bin/dep; \
    elif [ "${IMAGE_OS}" = "linux" -a "${IMAGE_ARCH}" = "arm64" ]; then \
	wget https://github.com/golang/dep/releases/download/v${DEP_VERSION}/dep-linux-arm64 -O /usr/local/bin/dep; \
    fi && \
    chmod +x /usr/local/bin/dep

####################################################################################################
# argoexec-base
# Used as the base for both the release and development version of argoexec
####################################################################################################
FROM debian:10.3-slim as argoexec-base

ARG IMAGE_OS=linux
ARG IMAGE_ARCH=amd64

# NOTE: keep the version synced with https://storage.googleapis.com/kubernetes-release/release/stable.txt
ENV KUBECTL_VERSION=1.15.1
ENV JQ_VERSION=1.6
RUN apt-get update && \
    apt-get --no-install-recommends install -y curl procps git apt-utils apt-transport-https ca-certificates tar mime-support && \
    apt-get clean \
    && rm -rf \
        /var/lib/apt/lists/* \
        /tmp/* \
        /var/tmp/* \
        /usr/share/man \
        /usr/share/doc \
        /usr/share/doc-base && \
    if [ "${IMAGE_OS}" = "linux" -a "${IMAGE_ARCH}" = "amd64" ]; then \
	curl -L -o /usr/local/bin/kubectl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl; \
    elif [ "${IMAGE_OS}" = "linux" -a "${IMAGE_ARCH}" = "arm64" ]; then \
	curl -L -o /usr/local/bin/kubectl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/arm64/kubectl; \
    fi && \
    chmod +x /usr/local/bin/kubectl && \
    curl -L -o /usr/local/bin/jq -LO https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64 && \
    chmod +x /usr/local/bin/jq
COPY hack/ssh_known_hosts /etc/ssh/ssh_known_hosts
COPY --from=builder /usr/local/bin/docker /usr/local/bin/

####################################################################################################

FROM node:11.15.0 as argo-ui

ADD ["ui", "."]

RUN yarn install --frozen-lockfile --ignore-optional --non-interactive
RUN yarn build

####################################################################################################
# Argo Build stage which performs the actual build of Argo binaries
####################################################################################################
FROM builder as argo-build

ARG IMAGE_OS=linux
ARG IMAGE_ARCH=amd64

# Perform the build
WORKDIR /go/src/github.com/argoproj/argo
COPY . .
# check we can use Git
RUN git rev-parse HEAD
COPY --from=argo-ui node_modules ui/node_modules
RUN mkdir -p ui/dist
COPY --from=argo-ui dist/app ui/dist/app
# stop make from trying to re-build this without yarn installed
RUN touch ui/dist/app
# fail the build if we are "dirty"
RUN git diff --exit-code
RUN if [ "${IMAGE_OS}" = "linux" -a "${IMAGE_ARCH}" = "amd64" ]; then \
	make dist/argo-linux-amd64 dist/workflow-controller-linux-amd64 dist/argoexec-linux-amd64; \
    elif [ "${IMAGE_OS}" = "linux" -a "${IMAGE_ARCH}" = "arm64" ]; then \
	make dist/argo-linux-arm64 dist/workflow-controller-linux-arm64 dist/argoexec-linux-arm64; \
    fi 

####################################################################################################
# argoexec
####################################################################################################
FROM argoexec-base as argoexec
COPY --from=argo-build /go/src/github.com/argoproj/argo/dist/argoexec-linux-* /usr/local/bin/argoexec
ENTRYPOINT [ "argoexec" ]

####################################################################################################
# workflow-controller
####################################################################################################
FROM scratch as workflow-controller
# Add timezone data
COPY --from=argo-build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=argo-build /go/src/github.com/argoproj/argo/dist/workflow-controller-linux-* /bin/workflow-controller
ENTRYPOINT [ "workflow-controller" ]

####################################################################################################
# argocli
####################################################################################################
FROM scratch as argocli
COPY --from=argoexec-base /etc/ssh/ssh_known_hosts /etc/ssh/ssh_known_hosts
COPY --from=argoexec-base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=argo-build /go/src/github.com/argoproj/argo/dist/argo-linux-* /bin/argo
ENTRYPOINT [ "argo" ]
