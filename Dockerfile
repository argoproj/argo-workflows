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
WORKDIR /go/src/github.com/argoproj/argo-server
COPY . .
# check we can use Git
RUN git rev-parse HEAD

# cli image
RUN mkdir -p ui/dist
COPY --from=argo-ui ui/dist/app ui/dist/app
# stop make from trying to re-build this without yarn installed
RUN touch ui/dist/node_modules.marker
RUN touch ui/dist/app/index.html
RUN . hack/image_arch.sh && make argo-server.crt argo-server.key dist/argo-${IMAGE_OS}-${IMAGE_ARCH}
RUN . hack/image_arch.sh && ./dist/argo-${IMAGE_OS}-${IMAGE_ARCH} version 2>&1 | grep clean


####################################################################################################
# argocli
####################################################################################################
FROM scratch as argocli
USER 8737
ARG IMAGE_OS=linux
COPY hack/ssh_known_hosts /etc/ssh/ssh_known_hosts
COPY hack/nsswitch.conf /etc/nsswitch.conf
COPY --from=argo-build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=argo-build --chown=8737 /go/src/github.com/argoproj/argo-server/argo-server.crt argo-server.crt
COPY --from=argo-build --chown=8737 /go/src/github.com/argoproj/argo-server/argo-server.key argo-server.key
COPY --from=argo-build /go/src/github.com/argoproj/argo-server/dist/argo-${IMAGE_OS}-* /bin/argo
ENTRYPOINT [ "argo" ]
