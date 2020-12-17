####################################################################################################
# Builder image
# Initial stage which pulls prepares build dependencies and CLI tooling we need for our final image
# Also used as the image in CI jobs so needs all dependencies
####################################################################################################
FROM golang:1.13.4 as builder

ARG IMAGE_OS=linux

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

RUN if [ "${IMAGE_OS}" = "linux" ]; then \
        export IMAGE_ARCH=`uname -m`; \
        if [ "${IMAGE_ARCH}" = "ppc64le" ] ||[ "${IMAGE_ARCH}" = "s390x" ]; then \
	        wget -O docker.tgz https://download.docker.com/${IMAGE_OS}/static/${DOCKER_CHANNEL}/${IMAGE_ARCH}/docker-18.06.3-ce.tgz; \
        else \
            wget -O docker.tgz https://download.docker.com/${IMAGE_OS}/static/${DOCKER_CHANNEL}/${IMAGE_ARCH}/docker-${DOCKER_VERSION}.tgz; \
        fi \
    fi && \
    tar --extract --file docker.tgz --strip-components 1 --directory /usr/local/bin/ && \
    rm docker.tgz

####################################################################################################
# argoexec-base
# Used as the base for both the release and development version of argoexec
####################################################################################################
FROM debian:10.3-slim as argoexec-base

ARG IMAGE_OS=linux

# NOTE: kubectl version should be one minor version less than https://storage.googleapis.com/kubernetes-release/release/stable.txt
ENV KUBECTL_VERSION=1.18.8
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
        /usr/share/doc-base

ADD hack/recurl.sh .
ADD hack/image_arch.sh .
RUN . ./image_arch.sh && ./recurl.sh /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/${IMAGE_OS}/${IMAGE_ARCH}/kubectl
RUN ./recurl.sh /usr/local/bin/jq https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64
RUN rm recurl.sh

COPY hack/ssh_known_hosts /etc/ssh/ssh_known_hosts
COPY hack/nsswitch.conf /etc/nsswitch.conf
COPY --from=builder /usr/local/bin/docker /usr/local/bin/

####################################################################################################

FROM node:14.0.0 as argo-ui

ADD ["ui", "ui"]
ADD ["api", "api"]

RUN JOBS=max yarn --cwd ui install --network-timeout 1000000
RUN JOBS=max yarn --cwd ui build

####################################################################################################
# Argo Build stage which performs the actual build of Argo binaries
####################################################################################################
FROM builder as argo-build

ARG IMAGE_OS=linux

# Perform the build
WORKDIR /go/src/github.com/argoproj/argo
COPY . .
# check we can use Git
RUN git rev-parse HEAD

# controller image
RUN . hack/image_arch.sh && make dist/workflow-controller-${IMAGE_OS}-${IMAGE_ARCH}
RUN . hack/image_arch.sh && ./dist/workflow-controller-${IMAGE_OS}-${IMAGE_ARCH} version | grep clean

# executor image
RUN . hack/image_arch.sh && make dist/argoexec-${IMAGE_OS}-${IMAGE_ARCH}
RUN . hack/image_arch.sh && ./dist/argoexec-${IMAGE_OS}-${IMAGE_ARCH} version | grep clean

# cli image
RUN mkdir -p ui/dist
COPY --from=argo-ui ui/dist/app ui/dist/app
# stop make from trying to re-build this without yarn installed
RUN touch ui/dist/node_modules.marker
RUN touch ui/dist/app/index.html
RUN . hack/image_arch.sh && make argo-server.crt argo-server.key dist/argo-${IMAGE_OS}-${IMAGE_ARCH}
RUN . hack/image_arch.sh && ./dist/argo-${IMAGE_OS}-${IMAGE_ARCH} version 2>&1 | grep clean

####################################################################################################
# argoexec
####################################################################################################
FROM argoexec-base as argoexec
ARG IMAGE_OS=linux
COPY --from=argo-build /go/src/github.com/argoproj/argo/dist/argoexec-${IMAGE_OS}-* /usr/local/bin/argoexec
ENTRYPOINT [ "argoexec" ]

####################################################################################################
# workflow-controller
####################################################################################################
FROM scratch as workflow-controller
USER 8737
ARG IMAGE_OS=linux
# Add timezone data
COPY --from=argo-build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=argo-build /go/src/github.com/argoproj/argo/dist/workflow-controller-${IMAGE_OS}-* /bin/workflow-controller
ENTRYPOINT [ "workflow-controller" ]

####################################################################################################
# argocli
####################################################################################################
FROM scratch as argocli
USER 8737
ARG IMAGE_OS=linux
COPY --from=argoexec-base /etc/ssh/ssh_known_hosts /etc/ssh/ssh_known_hosts
COPY --from=argoexec-base /etc/nsswitch.conf /etc/nsswitch.conf
COPY --from=argoexec-base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=argo-build --chown=8737 /go/src/github.com/argoproj/argo/argo-server.crt argo-server.crt
COPY --from=argo-build --chown=8737 /go/src/github.com/argoproj/argo/argo-server.key argo-server.key
COPY --from=argo-build /go/src/github.com/argoproj/argo/dist/argo-${IMAGE_OS}-* /bin/argo
ENTRYPOINT [ "argo" ]
